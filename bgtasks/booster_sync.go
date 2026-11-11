// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"popplio/boosters"
	"popplio/db"
	"popplio/notifications"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func boosterPerksURL() pgtype.Text {
	return pgtype.Text{String: state.Config.Sites.Frontend + "/shop/booster", Valid: true}
}

func BoosterSync(ctx context.Context) error {
	ids, err := boosters.List()

	if errors.Is(err, boosters.ErrNotReady) {
		state.Logger.Info("booster_sync: Discord member cache not ready, skipping this run")
		return nil
	}

	if err != nil {
		return err
	}

	q := db.New(state.Pool)

	current, err := q.ListBoosting(ctx)

	if err != nil {
		return fmt.Errorf("listing current boosters: %w", err)
	}

	missing := []string{}

	for _, id := range current {
		if !slices.Contains(ids, id) {
			missing = append(missing, id)
		}
	}

	ids = append(ids, boosters.StillBoosting(missing)...)

	recentlyStopped, err := q.ListRecentlyStoppedBoosters(ctx)

	if err != nil {
		return fmt.Errorf("listing recently stopped boosters: %w", err)
	}

	started, err := q.StartBoosting(ctx, ids)

	if err != nil {
		return fmt.Errorf("recording new boosters: %w", err)
	}

	stopped, err := q.StopBoosting(ctx, ids)

	if err != nil {
		return fmt.Errorf("recording ended boosts: %w", err)
	}

	newBoosters := slices.DeleteFunc(slices.Clone(started), func(id string) bool {
		return slices.Contains(recentlyStopped, id)
	})

	notifyBoosters(ctx, q, newBoosters, types.Alert{
		Type:     types.AlertTypeSuccess,
		Title:    "Thanks for boosting!",
		Message:  "You now get a free month of Bronze premium every 30 days, a free extra Auto-vote slot and the Server Booster badge. Pick the listing your premium goes to.",
		URL:      boosterPerksURL(),
		Category: types.AlertCategoryPayments,
	})

	notifyBoosters(ctx, q, stopped, types.Alert{
		Type:     types.AlertTypeInfo,
		Title:    "Your booster perks have ended",
		Message:  "You're no longer boosting the Omniplex server, so your monthly premium and extra Auto-vote slot have stopped. Premium you already got stays until it runs out. Thanks for your support!",
		URL:      boosterPerksURL(),
		Category: types.AlertCategoryPayments,
	})

	if len(started) > 0 || len(stopped) > 0 {
		state.Logger.Info("booster_sync: updated boosters", zap.Int("started", len(started)), zap.Int("stopped", len(stopped)))
	}

	due, err := q.ListDueBoosterRenewals(ctx)

	if err != nil {
		return fmt.Errorf("listing booster renewals: %w", err)
	}

	for _, row := range due {
		RenewBoosterPremium(ctx, row.UserID, row.TargetType, row.TargetID)
	}

	return nil
}

func notifyBoosters(ctx context.Context, q *db.Queries, ids []string, alert types.Alert) {
	if len(ids) == 0 {
		return
	}

	users, err := q.ListExistingUsers(ctx, ids)

	if err != nil {
		state.Logger.Warn("booster_sync: failed to look up users to notify", zap.Error(err))
		return
	}

	for _, id := range users {
		if err := notifications.PushNotification(id, alert); err != nil {
			state.Logger.Warn("booster_sync: failed to notify a booster", zap.Error(err), zap.String("user_id", id))
		}
	}
}

func RenewBoosterPremium(ctx context.Context, userID, targetType, targetID string) bool {
	q := db.New(state.Pool)
	hours, err := assets.ApplyBoosterPremium(ctx, userID, targetType, targetID, time.Now())

	if err != nil {
		var targetErr *assets.TargetError

		if !errors.As(err, &targetErr) {
			state.Logger.Error("booster_sync: failed to apply booster premium", zap.Error(err), zap.String("user_id", userID))
			return false
		}

		msg := "Your booster premium couldn't be applied because " + targetErr.Reason + ". Pick another listing."
		_, setErr := q.SetBoosterError(ctx, db.SetBoosterErrorParams{UserID: userID, LastError: pgtype.Text{String: msg, Valid: true}})

		if setErr == nil {
			if err := notifications.PushNotification(userID, types.Alert{
				Type:     types.AlertTypeWarning,
				Title:    "Booster premium wasn't applied",
				Message:  msg,
				URL:      boosterPerksURL(),
				Category: types.AlertCategoryPayments,
			}); err != nil {
				state.Logger.Warn("booster_sync: failed to send renewal error", zap.Error(err), zap.String("user_id", userID))
			}
		}

		return false
	}

	if hours == 0 {
		return false
	}

	_, _ = q.SetBoosterError(ctx, db.SetBoosterErrorParams{UserID: userID})

	name := "your " + targetType

	if info, err := votes.GetEntityInfo(ctx, state.Pool, targetID, targetType); err == nil {
		name = info.Name
	}

	if err := notifications.PushNotification(userID, types.Alert{
		Type:     types.AlertTypeSuccess,
		Title:    "Booster premium applied",
		Message:  name + " got 30 days of Bronze premium as a thank you for boosting.",
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + targetType + "s/" + targetID, Valid: true},
		Category: types.AlertCategoryPayments,
	}); err != nil {
		state.Logger.Warn("booster_sync: failed to send renewal notice", zap.Error(err), zap.String("user_id", userID))
	}

	return true
}
