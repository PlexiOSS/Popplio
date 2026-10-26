// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"popplio/badgerules"
	"popplio/db"
	"popplio/linkedroles"
	"popplio/state"

	"go.uber.org/zap"
)

var badgeTargetTypes = []string{"user", "bot", "server", "team"}

func BadgeSync(ctx context.Context) error {
	q := db.New(state.Pool)

	badges, err := q.ListAutoBadges(ctx)

	if err != nil {
		return fmt.Errorf("listing automatic badges: %w", err)
	}

	for _, b := range badges {
		var rule badgerules.Rule

		raw, _ := json.Marshal(b.AutoRule)

		if err := json.Unmarshal(raw, &rule); err != nil {
			state.Logger.Warn("badge_sync: unreadable rule, skipping", zap.String("badge", b.ID), zap.Error(err))
			continue
		}

		kind, ok := badgerules.KindOf(rule.Type)

		if !ok {
			state.Logger.Warn("badge_sync: unknown rule type, skipping", zap.String("badge", b.ID), zap.String("rule", rule.Type))
			continue
		}

		allowed := make(map[string]bool, len(b.TargetTypes))

		for _, t := range b.TargetTypes {
			allowed[t] = true
		}

		for _, targetType := range badgeTargetTypes {
			if !allowed[targetType] {
				continue
			}

			ids := []string{}

			if kind.Supports(targetType) {
				ids, err = badgerules.Qualifying(ctx, rule, targetType)

				if errors.Is(err, badgerules.ErrDiscordNotReady) {
					state.Logger.Info("badge_sync: Discord member cache not ready, leaving Discord badges alone this run", zap.String("badge", b.ID))
					continue
				}

				if err != nil {
					return fmt.Errorf("%s/%s: listing qualifying entities: %w", b.ID, targetType, err)
				}
			}

			awarded, err := q.AwardSystemBadge(ctx, db.AwardSystemBadgeParams{
				TargetType: targetType,
				BadgeID:    b.ID,
				Reason:     kind.Description,
				TargetIds:  ids,
			})

			if err != nil {
				return fmt.Errorf("%s/%s: awarding: %w", b.ID, targetType, err)
			}

			revoked, err := q.RevokeSystemBadge(ctx, db.RevokeSystemBadgeParams{
				TargetType: targetType,
				BadgeID:    b.ID,
				TargetIds:  ids,
			})

			if err != nil {
				return fmt.Errorf("%s/%s: revoking: %w", b.ID, targetType, err)
			}

			if awarded > 0 || revoked > 0 {
				state.Logger.Info("badge_sync: updated automatic badges",
					zap.String("badge", b.ID),
					zap.String("targetType", targetType),
					zap.Int64("awarded", awarded),
					zap.Int64("revoked", revoked))
			}
		}
	}

	orphans, err := q.RevokeOrphanSystemBadges(ctx)

	if err != nil {
		return fmt.Errorf("revoking badges whose rule was removed: %w", err)
	}

	if orphans > 0 {
		state.Logger.Info("badge_sync: removed automatic badges whose rule or target type was removed", zap.Int64("revoked", orphans))
	}

	return linkedroles.SyncAll(ctx)
}
