package autovote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"popplio/audit"
	"popplio/db"
	"popplio/promo"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v75"
	portal "github.com/stripe/stripe-go/v75/billingportal/session"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"github.com/stripe/stripe-go/v75/subscription"
	"go.uber.org/zap"
)

const (
	PriceCents  = 299
	MaxTargets  = 3
	metadataKey = "kind"
	metadataVal = "auto_vote"
)

var ErrNoSubscription = errors.New("no auto-vote subscription")

func Active(status string, periodEnd pgtype.Timestamptz) bool {
	if status != string(stripe.SubscriptionStatusActive) && status != string(stripe.SubscriptionStatusTrialing) {
		return false
	}

	return !periodEnd.Valid || periodEnd.Time.After(time.Now())
}

func InSchedule(days []int16, startMinute, endMinute int, timezone string, now time.Time) bool {
	loc, err := time.LoadLocation(timezone)

	if err != nil {
		loc = time.UTC
	}

	local := now.In(loc)

	if len(days) > 0 && !slices.Contains(days, int16(local.Weekday())) {
		return false
	}

	minute := local.Hour()*60 + local.Minute()

	return minute >= startMinute && minute < endMinute
}

func Target(ctx context.Context, row db.AutoVoteTarget) types.AutoVoteTarget {
	target := types.AutoVoteTarget{
		TargetType:  row.TargetType,
		TargetID:    row.TargetID,
		Name:        row.TargetID,
		Days:        make([]int, len(row.Days)),
		StartMinute: int(row.StartMinute),
		EndMinute:   int(row.EndMinute),
		Timezone:    row.Timezone,
		InSchedule:  InSchedule(row.Days, int(row.StartMinute), int(row.EndMinute), row.Timezone, time.Now()),
	}

	for i, day := range row.Days {
		target.Days[i] = int(day)
	}

	if info, err := votes.GetEntityInfo(ctx, state.Pool, row.TargetID, row.TargetType); err == nil {
		target.Name = info.Name
		target.Avatar = info.Avatar
	}

	if row.LastVotedAt.Valid {
		at := row.LastVotedAt.Time
		target.LastVotedAt = &at
	}

	if row.LastSkipReason.Valid {
		reason := row.LastSkipReason.String
		target.LastSkipReason = &reason
	}

	check, err := votes.EntityVoteCheck(ctx, state.Pool, row.UserID, row.TargetID, row.TargetType)

	if err != nil {
		state.Logger.Warn("Failed to check auto-vote cooldown", zap.Error(err), zap.String("userId", row.UserID), zap.String("targetId", row.TargetID))
	} else if check.HasVoted && len(check.ValidVotes) > 0 {
		until := check.ValidVotes[0].CreatedAt.Add(time.Duration(check.VoteInfo.VoteTime) * time.Hour)
		target.CooldownUntil = &until
	}

	return target
}

func Checkout(userID string, discounts []*stripe.CheckoutSessionDiscountParams) (*stripe.CheckoutSession, error) {
	if state.Config.Meta.StripeSecretKey == "" {
		return nil, errors.New("stripe is not configured")
	}

	metadata := map[string]string{metadataKey: metadataVal, "user_id": userID}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(string(stripe.CurrencyUSD)),
					UnitAmount: stripe.Int64(PriceCents),
					Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
					},
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("Omniplex Auto-vote"),
						Description: stripe.String("Automatically vote for up to 3 bots or servers whenever your cooldown ends."),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String("autovote:" + userID),
		Metadata:          metadata,
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		},
		SuccessURL: stripe.String(state.Config.Sites.Frontend + "/dashboard?tab=auto-vote&subscribed=1"),
		CancelURL:  stripe.String(state.Config.Sites.Frontend + "/dashboard?tab=auto-vote"),
		Discounts:  discounts,
	}

	return session.New(params)
}

func PortalURL(customerID string) (string, error) {
	s, err := portal.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(state.Config.Sites.Frontend + "/dashboard?tab=auto-vote"),
	})

	if err != nil {
		return "", err
	}

	return s.URL, nil
}

func HandleStripeEvent(ctx context.Context, event stripe.Event) (bool, uapi.HttpResponse) {
	switch event.Type {
	case "checkout.session.completed":
		var s stripe.CheckoutSession

		if err := jsonimpl.Unmarshal(event.Data.Raw, &s); err != nil {
			return false, uapi.HttpResponse{}
		}

		if s.Mode != stripe.CheckoutSessionModeSubscription || s.Metadata[metadataKey] != metadataVal {
			return false, uapi.HttpResponse{}
		}

		if s.Subscription == nil || s.Customer == nil {
			return true, failed("checkout session has no subscription or customer", nil)
		}

		sub, err := subscription.Get(s.Subscription.ID, nil)

		if err != nil {
			return true, failed("fetching subscription", err)
		}

		if err := db.New(state.Pool).UpsertAutoVoteSubscription(ctx, db.UpsertAutoVoteSubscriptionParams{
			UserID:               s.Metadata["user_id"],
			StripeCustomerID:     s.Customer.ID,
			StripeSubscriptionID: sub.ID,
			Status:               string(sub.Status),
			CurrentPeriodEnd:     periodEnd(sub.CurrentPeriodEnd),
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}); err != nil {
			return true, failed("saving subscription", err)
		}

		promo.Complete(ctx, s.ID)

		audit.Record(audit.Event{
			Source:     audit.SourceStripe,
			ActorType:  audit.ActorUser,
			ActorID:    s.Metadata["user_id"],
			Action:     "autovote.subscribed",
			TargetType: "user",
			TargetID:   s.Metadata["user_id"],
			Data:       map[string]any{"session": s.ID, "subscription": sub.ID, "status": string(sub.Status)},
		})

		return true, ok("auto-vote subscription started")
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription

		if err := jsonimpl.Unmarshal(event.Data.Raw, &sub); err != nil {
			return false, uapi.HttpResponse{}
		}

		if sub.Metadata[metadataKey] != metadataVal {
			return true, ok("not an auto-vote subscription")
		}

		if _, err := db.New(state.Pool).UpdateAutoVoteSubscriptionByStripeID(ctx, db.UpdateAutoVoteSubscriptionByStripeIDParams{
			StripeSubscriptionID: sub.ID,
			Status:               string(sub.Status),
			CurrentPeriodEnd:     periodEnd(sub.CurrentPeriodEnd),
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}); err != nil {
			return true, failed("updating subscription", err)
		}

		audit.Record(audit.Event{
			Source:     audit.SourceStripe,
			ActorType:  audit.ActorUser,
			ActorID:    sub.Metadata["user_id"],
			Action:     "autovote.subscription_" + strings.TrimPrefix(string(event.Type), "customer.subscription."),
			TargetType: "user",
			TargetID:   sub.Metadata["user_id"],
			Data:       map[string]any{"subscription": sub.ID, "status": string(sub.Status), "cancel_at_period_end": sub.CancelAtPeriodEnd},
		})

		return true, ok("auto-vote subscription updated")
	}

	return false, uapi.HttpResponse{}
}

func periodEnd(unix int64) pgtype.Timestamptz {
	if unix == 0 {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: time.Unix(unix, 0), Valid: true}
}

func ok(msg string) uapi.HttpResponse {
	return uapi.HttpResponse{Status: http.StatusOK, Data: msg}
}

func failed(msg string, err error) uapi.HttpResponse {
	state.Logger.Error("Auto-vote Stripe webhook failed: "+msg, zap.Error(err))
	return uapi.HttpResponse{Status: http.StatusInternalServerError, Data: fmt.Sprintf("auto-vote: %s", msg)}
}
