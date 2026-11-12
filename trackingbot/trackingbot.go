// Copyright (C) 2026 NodeByte LTD

package trackingbot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/impls"
	arcadiatypes "popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const HideAfter = 14 * 24 * time.Hour

const (
	noticeNone   int16 = 0
	noticeWarned int16 = 1
	noticeHidden int16 = 2
)

type Phase int

const (
	PhaseOK Phase = iota
	PhaseWarn
	PhaseHide
)

func NextPhase(missingSince pgtype.Timestamptz, notice int16, now time.Time) Phase {
	if !missingSince.Valid {
		return PhaseOK
	}

	if notice < noticeHidden && now.Sub(missingSince.Time) >= HideAfter {
		return PhaseHide
	}

	if notice < noticeWarned {
		return PhaseWarn
	}

	return PhaseOK
}

func Missing(ctx context.Context, serverID string) error {
	q := db.New(state.Pool)

	rows, err := q.MarkTrackingBotMissing(ctx, serverID)

	if err != nil || rows == 0 {
		return err
	}

	name, err := serverName(ctx, serverID)

	if err != nil {
		return err
	}

	return warn(ctx, serverID, name)
}

func Present(ctx context.Context, serverID string) error {
	row, err := db.New(state.Pool).ClearTrackingBotMissing(ctx, serverID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	if row.PreviousNotice >= noticeHidden {
		notify(ctx, serverID, types.Alert{
			Type:    types.AlertTypeSuccess,
			Title:   row.Name + " is listed again",
			Message: "The Omniplex Servers bot is back, so your server shows in browse and search again.",
		})
	}

	return nil
}

type Checker func(ctx context.Context, serverID string) (bool, error)

func Sweep(ctx context.Context, inServer Checker) error {
	q := db.New(state.Pool)

	servers, err := q.ListTrackingBotServers(ctx)

	if err != nil {
		return fmt.Errorf("listing servers: %w", err)
	}

	now := time.Now()
	var errs []error

	for _, s := range servers {
		if err := ctx.Err(); err != nil {
			return err
		}

		present, err := inServer(ctx, s.ServerID)

		if err != nil {
			state.Logger.Warn("trackingbot: presence check failed, skipping", zap.Error(err), zap.String("server_id", s.ServerID))
			continue
		}

		if present {
			if s.TrackingBotMissingSince.Valid {
				errs = append(errs, Present(ctx, s.ServerID))
			}

			continue
		}

		if !s.TrackingBotMissingSince.Valid {
			errs = append(errs, Missing(ctx, s.ServerID))
			continue
		}

		switch NextPhase(s.TrackingBotMissingSince, s.TrackingBotNotice, now) {
		case PhaseWarn:
			errs = append(errs, warn(ctx, s.ServerID, s.Name))
		case PhaseHide:
			errs = append(errs, hide(ctx, s.ServerID, s.Name))
		}
	}

	return errors.Join(errs...)
}

func warn(ctx context.Context, serverID, name string) error {
	notify(ctx, serverID, types.Alert{
		Type:    types.AlertTypeWarning,
		Title:   "Omniplex Servers left " + name,
		Message: "Add the bot back within 14 days to keep " + name + " in browse and search. Until then, member counts and vote messages won't update.",
	})

	return db.New(state.Pool).SetTrackingBotNotice(ctx, db.SetTrackingBotNoticeParams{ServerID: serverID, TrackingBotNotice: noticeWarned})
}

func hide(ctx context.Context, serverID, name string) error {
	q := db.New(state.Pool)

	if _, err := q.HideServerForTrackingBot(ctx, serverID); err != nil {
		return err
	}

	notify(ctx, serverID, types.Alert{
		Type:    types.AlertTypeWarning,
		Title:   name + " is hidden from browse and search",
		Message: "The Omniplex Servers bot has been missing for 14 days. Add it back and your listing comes back by itself.",
	})

	return q.SetTrackingBotNotice(ctx, db.SetTrackingBotNoticeParams{ServerID: serverID, TrackingBotNotice: noticeHidden})
}

func serverName(ctx context.Context, serverID string) (string, error) {
	row, err := db.New(state.Pool).GetServerNameAndShort(ctx, serverID)

	if err != nil {
		return "", err
	}

	return row.Name, nil
}

func notify(ctx context.Context, serverID string, alert types.Alert) {
	managers, err := impls.GetEntityManagers(ctx, arcadiatypes.TargetTypeServer, serverID)

	if err != nil {
		state.Logger.Warn("trackingbot: failed to find owners", zap.Error(err), zap.String("server_id", serverID))
		return
	}

	alert.URL = pgtype.Text{String: fmt.Sprintf("%s/dashboard/servers/%s", state.Config.Sites.Frontend, serverID), Valid: true}
	alert.Category = types.AlertCategoryBotServerReviews
	alert.Priority = types.AlertPriorityMedium
	alert.AlertData = map[string]any{"target_type": "server", "target_id": serverID}

	impls.NotifyOwners(managers.All(), alert)
}
