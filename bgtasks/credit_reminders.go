// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/impls"
	arcadiatypes "popplio/arcadia/types"
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	creditReminderWindow   = 3 * 24 * time.Hour
	creditReminderMinVotes = 10
)

type creditTally struct {
	toRedeem    map[string]bool
	withCredits map[string]bool
}

func listings(n int) string {
	if n == 1 {
		return "1 of your listings has"
	}

	return fmt.Sprintf("%d of your listings have", n)
}

func CreditReminderMessage(resetAt time.Time, toRedeem, withCredits int) string {
	day := resetAt.UTC().Format("2 January")

	switch {
	case toRedeem > 0 && withCredits > 0:
		return fmt.Sprintf("Votes reset on %s. %s votes you can still turn into credits, and %d %s credits waiting to be spent.", day, listings(toRedeem), withCredits, map[bool]string{true: "has", false: "have"}[withCredits == 1])
	case toRedeem > 0:
		return fmt.Sprintf("Votes reset on %s. %s votes you can still turn into credits.", day, listings(toRedeem))
	case withCredits > 0:
		return fmt.Sprintf("%s credits waiting to be spent in the Credit shop.", listings(withCredits))
	}

	return ""
}

func CreditReminders(ctx context.Context) error {
	q := db.New(state.Pool)

	next, err := q.GetNextVoteReset(ctx)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("finding the next vote reset: %w", err)
	}

	if !next.Valid {
		return nil
	}

	until := time.Until(next.Time)

	if until <= 0 || until > creditReminderWindow {
		return nil
	}

	cheapest, err := q.GetCheapestShopItemPrice(ctx)

	if err != nil {
		return fmt.Errorf("finding the cheapest shop item: %w", err)
	}

	redeemable, err := q.ListRedeemableVoteTotals(ctx, creditReminderMinVotes)

	if err != nil {
		return fmt.Errorf("listing redeemable votes: %w", err)
	}

	unspent, err := q.ListUnspentCreditTotals(ctx, max(cheapest, 1))

	if err != nil {
		return fmt.Errorf("listing unspent credits: %w", err)
	}

	tallies := map[string]*creditTally{}

	add := func(targetType, targetID string, redeem bool) {
		managerType := arcadiatypes.TargetTypeBot

		if targetType == "server" {
			managerType = arcadiatypes.TargetTypeServer
		}

		managers, err := impls.GetEntityManagers(ctx, managerType, targetID)

		if err != nil {
			state.Logger.Warn("credit_reminders: failed to find owners", zap.Error(err), zap.String("target_id", targetID))
			return
		}

		key := targetType + ":" + targetID

		for _, user := range managers.All() {
			t, ok := tallies[user]

			if !ok {
				t = &creditTally{toRedeem: map[string]bool{}, withCredits: map[string]bool{}}
				tallies[user] = t
			}

			if redeem {
				t.toRedeem[key] = true
			} else {
				t.withCredits[key] = true
			}
		}
	}

	for _, r := range redeemable {
		add(r.TargetType, r.TargetID, true)
	}

	for _, r := range unspent {
		add(r.TargetType, r.TargetID, false)
	}

	period := next.Time.UTC().Format("2006-01-02")
	sent := 0

	for user, t := range tallies {
		message := CreditReminderMessage(next.Time, len(t.toRedeem), len(t.withCredits))

		if message == "" {
			continue
		}

		if _, err := q.MarkCreditReminderSent(ctx, db.MarkCreditReminderSentParams{UserID: user, Period: period}); errors.Is(err, pgx.ErrNoRows) {
			continue
		} else if err != nil {
			return fmt.Errorf("recording a credit reminder: %w", err)
		}

		if err := notifications.PushNotification(user, types.Alert{
			Type:     types.AlertTypeInfo,
			Title:    "Your vote credits",
			Message:  message,
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/shop/items", Valid: true},
			Category: types.AlertCategoryShop,
		}); err != nil {
			state.Logger.Warn("credit_reminders: failed to notify", zap.Error(err), zap.String("user_id", user))
			continue
		}

		sent++
	}

	if sent > 0 {
		state.Logger.Info("credit_reminders: sent reminders", zap.Int("users", sent), zap.String("period", period))
	}

	return nil
}
