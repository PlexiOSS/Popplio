// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"popplio/audit"
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/charge"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"github.com/stripe/stripe-go/v75/invoice"
	"github.com/stripe/stripe-go/v75/subscription"
	"go.uber.org/zap"
)

const (
	RevokeRefunded = "refunded"
	RevokeDisputed = "disputed"
)

var ErrNotFulfilled = errors.New("no fulfilled purchase matches this payment")

type RevokeResult struct {
	AlreadyRevoked  bool
	PremiumRemoved  bool
	PremiumEnded    bool
	HoursRemoved    int32
	CreditsReturned int32
	ProductID       string
	UserID          string
	ForType         string
	ForID           string
}

func RevokePurchase(ctx context.Context, provider, externalID, reason string, returnCredits bool) (*RevokeResult, error) {
	res, err := revoke(ctx, provider, externalID, reason, returnCredits)

	if err != nil || res.AlreadyRevoked {
		return res, err
	}

	audit.Record(audit.Event{
		Source:     provider,
		ActorType:  audit.ActorUser,
		ActorID:    res.UserID,
		Action:     "payment." + reason,
		TargetType: res.ForType,
		TargetID:   res.ForID,
		Data: map[string]any{
			"external_id":      externalID,
			"product":          res.ProductID,
			"premium_removed":  res.PremiumRemoved,
			"premium_ended":    res.PremiumEnded,
			"hours_removed":    res.HoursRemoved,
			"credits_returned": res.CreditsReturned,
		},
	})

	message := "A payment for premium on " + res.ForType + " " + res.ForID + " was " + reason + ", so "

	switch {
	case res.PremiumRemoved && res.PremiumEnded:
		message += "its premium has been removed."
	case res.PremiumRemoved:
		message += "the " + durationLabel(res.HoursRemoved) + " of premium it added were removed. Premium from other purchases stays."
	default:
		message += "it has been marked as refunded. Premium from that purchase had already ended or been replaced."
	}

	if res.CreditsReturned > 0 {
		message += fmt.Sprintf(" The %d credits used on it were returned.", res.CreditsReturned)
	}

	if err := notifications.PushNotification(res.UserID, types.Alert{
		Type:     types.AlertTypeWarning,
		Title:    "Premium payment " + reason,
		Message:  message,
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/support", Valid: true},
		Category: types.AlertCategoryPayments,
	}); err != nil {
		state.Logger.Warn("Failed to send revoked purchase alert", zap.Error(err), zap.String("user_id", res.UserID))
	}

	return res, nil
}

func revoke(ctx context.Context, provider, externalID, reason string, returnCredits bool) (*RevokeResult, error) {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	f, err := q.GetPaymentFulfillment(ctx, db.GetPaymentFulfillmentParams{Provider: provider, ExternalID: externalID})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFulfilled
	}

	if err != nil {
		return nil, fmt.Errorf("reading fulfillment: %w", err)
	}

	res := &RevokeResult{UserID: f.UserID, ProductID: f.ProductID, ForType: f.ForType, ForID: f.ForID}

	marked, err := q.MarkPaymentFulfillmentRevoked(ctx, db.MarkPaymentFulfillmentRevokedParams{
		Provider:     provider,
		ExternalID:   externalID,
		RevokeReason: pgtype.Text{String: reason, Valid: true},
	})

	if err != nil {
		return nil, fmt.Errorf("marking fulfillment revoked: %w", err)
	}

	if marked == 0 {
		res.AlreadyRevoked = true
		return res, nil
	}

	var stillPremium bool

	switch f.ForType {
	case "bot":
		stillPremium, err = q.RevokeBotPremiumFrom(ctx, db.RevokeBotPremiumFromParams{Hours: f.PremiumHours, BotID: f.ForID, GrantedAt: f.FulfilledAt})
	case "server":
		stillPremium, err = q.RevokeServerPremiumFrom(ctx, db.RevokeServerPremiumFromParams{Hours: f.PremiumHours, ServerID: f.ForID, GrantedAt: f.FulfilledAt})
	default:
		err = pgx.ErrNoRows
	}

	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("removing premium: %w", err)
	default:
		res.PremiumRemoved = true
		res.PremiumEnded = !stillPremium
		res.HoursRemoved = f.PremiumHours.Int32
	}

	if returnCredits {
		hold, err := q.ReturnSpentPaymentCreditHold(ctx, db.ReturnSpentPaymentCreditHoldParams{Provider: provider, ExternalID: externalID})

		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return nil, fmt.Errorf("returning credits: %w", err)
		default:
			if _, err := q.InsertVoteRedeemLog(ctx, db.InsertVoteRedeemLogParams{
				TargetID:   hold.TargetID,
				TargetType: hold.TargetType,
				Credits:    hold.Credits,
			}); err != nil {
				return nil, fmt.Errorf("returning credits: %w", err)
			}

			res.CreditsReturned = hold.Credits
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return res, nil
}

func durationLabel(hours int32) string {
	switch {
	case hours <= 0:
		return "time"
	case hours%24 == 0 && hours/24 == 1:
		return "1 day"
	case hours%24 == 0:
		return fmt.Sprintf("%d days", hours/24)
	case hours == 1:
		return "1 hour"
	default:
		return fmt.Sprintf("%d hours", hours)
	}
}

func chargeData(ch *stripe.Charge) map[string]any {
	data := map[string]any{
		"charge":          ch.ID,
		"amount":          ch.Amount,
		"amount_refunded": ch.AmountRefunded,
		"currency":        ch.Currency,
	}

	if ch.PaymentIntent != nil {
		data["payment_intent"] = ch.PaymentIntent.ID
	}

	if ch.Invoice != nil {
		data["invoice"] = ch.Invoice.ID
	}

	return data
}

func webhookOK(msg string) uapi.HttpResponse {
	return uapi.HttpResponse{Status: http.StatusOK, Data: msg}
}

func webhookRetry(msg string, err error) uapi.HttpResponse {
	state.Logger.Error("Payment webhook event failed, the provider will retry", zap.Error(err), zap.String("step", msg))
	return uapi.HttpResponse{Status: http.StatusInternalServerError, Data: "Failed: " + msg}
}

func HandleStripeChargeEvent(ctx context.Context, event stripe.Event) (bool, uapi.HttpResponse) {
	var ch *stripe.Charge
	var reason string

	switch event.Type {
	case "charge.refunded":
		var c stripe.Charge

		if err := jsonimpl.Unmarshal(event.Data.Raw, &c); err != nil {
			return true, webhookRetry("reading charge", err)
		}

		ch, reason = &c, RevokeRefunded
	case "charge.dispute.created":
		var d stripe.Dispute

		if err := jsonimpl.Unmarshal(event.Data.Raw, &d); err != nil {
			return true, webhookRetry("reading dispute", err)
		}

		if d.Charge == nil {
			return true, webhookOK("dispute has no charge")
		}

		c, err := charge.Get(d.Charge.ID, nil)

		if err != nil {
			return true, webhookRetry("fetching disputed charge", err)
		}

		ch, reason = c, RevokeDisputed
	default:
		return false, uapi.HttpResponse{}
	}

	data := chargeData(ch)

	if reason == RevokeRefunded && !ch.Refunded {
		audit.Record(audit.Event{Source: audit.SourceStripe, ActorType: audit.ActorSystem, Action: "payment.partially_refunded", Data: data})
		return true, webhookOK("partial refund recorded, nothing revoked")
	}

	if ch.Invoice != nil && ch.Invoice.ID != "" {
		return true, handleSubscriptionCharge(ch, reason, data)
	}

	if ch.PaymentIntent == nil || ch.PaymentIntent.ID == "" {
		audit.Record(audit.Event{Source: audit.SourceStripe, ActorType: audit.ActorSystem, Action: "payment." + reason + "_unmatched", Data: data})
		return true, webhookOK("charge has no payment intent")
	}

	params := &stripe.CheckoutSessionListParams{PaymentIntent: stripe.String(ch.PaymentIntent.ID)}
	params.Limit = stripe.Int64(1)

	iter := session.List(params)

	if !iter.Next() {
		if err := iter.Err(); err != nil {
			return true, webhookRetry("finding checkout session", err)
		}

		audit.Record(audit.Event{Source: audit.SourceStripe, ActorType: audit.ActorSystem, Action: "payment." + reason + "_unmatched", Data: data})
		return true, webhookOK("no checkout session for this charge")
	}

	s := iter.CheckoutSession()

	if _, err := RevokePurchase(ctx, "stripe", s.ID, reason, reason == RevokeRefunded); err != nil {
		if errors.Is(err, ErrNotFulfilled) {
			data["session"] = s.ID
			audit.Record(audit.Event{Source: audit.SourceStripe, ActorType: audit.ActorSystem, Action: "payment." + reason + "_unmatched", Data: data})
			return true, webhookOK("checkout session was never fulfilled")
		}

		return true, webhookRetry("revoking purchase", err)
	}

	return true, webhookOK("purchase " + reason)
}

func handleSubscriptionCharge(ch *stripe.Charge, reason string, data map[string]any) uapi.HttpResponse {
	inv, err := invoice.Get(ch.Invoice.ID, nil)

	if err != nil {
		return webhookRetry("fetching invoice", err)
	}

	if inv.Subscription == nil || inv.Subscription.ID == "" {
		audit.Record(audit.Event{Source: audit.SourceStripe, ActorType: audit.ActorSystem, Action: "payment." + reason + "_unmatched", Data: data})
		return webhookOK("invoice has no subscription")
	}

	sub, err := subscription.Get(inv.Subscription.ID, nil)

	if err != nil {
		return webhookRetry("fetching subscription", err)
	}

	userID := sub.Metadata["user_id"]
	data["subscription"] = sub.ID

	if reason == RevokeDisputed && sub.Status != stripe.SubscriptionStatusCanceled {
		if _, err := subscription.Cancel(sub.ID, nil); err != nil {
			return webhookRetry("cancelling disputed subscription", err)
		}

		data["cancelled"] = true
	}

	audit.Record(audit.Event{
		Source:     audit.SourceStripe,
		ActorType:  audit.ActorUser,
		ActorID:    userID,
		Action:     "autovote.payment_" + reason,
		TargetType: "user",
		TargetID:   userID,
		Data:       data,
	})

	return webhookOK("subscription payment " + reason)
}
