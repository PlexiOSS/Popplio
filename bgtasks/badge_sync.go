// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"

	"popplio/db"
	"popplio/state"

	"go.uber.org/zap"
)

type systemBadgeRule struct {
	badgeID    string
	targetType string
	reason     string
	targets    func(ctx context.Context, q *db.Queries) ([]string, error)
}

var systemBadgeRules = []systemBadgeRule{
	{
		badgeID:    "partner",
		targetType: "bot",
		reason:     "An official Omniplex partner.",
		targets:    func(ctx context.Context, q *db.Queries) ([]string, error) { return q.GetPartnerBotIDs(ctx) },
	},
	{
		badgeID:    "veteran",
		targetType: "bot",
		reason:     "Listed on Omniplex for more than two years.",
		targets:    func(ctx context.Context, q *db.Queries) ([]string, error) { return q.GetVeteranBotIDs(ctx) },
	},
	{
		badgeID:    "veteran",
		targetType: "server",
		reason:     "Listed on Omniplex for more than two years.",
		targets:    func(ctx context.Context, q *db.Queries) ([]string, error) { return q.GetVeteranServerIDs(ctx) },
	},
}

func BadgeSync(ctx context.Context) error {
	q := db.New(state.Pool)

	for _, rule := range systemBadgeRules {
		ids, err := rule.targets(ctx, q)

		if err != nil {
			return fmt.Errorf("%s/%s: listing qualifying entities: %w", rule.badgeID, rule.targetType, err)
		}

		if ids == nil {
			ids = []string{}
		}

		awarded, err := q.AwardSystemBadge(ctx, db.AwardSystemBadgeParams{
			TargetType: rule.targetType,
			BadgeID:    rule.badgeID,
			Reason:     rule.reason,
			TargetIds:  ids,
		})

		if err != nil {
			return fmt.Errorf("%s/%s: awarding: %w", rule.badgeID, rule.targetType, err)
		}

		revoked, err := q.RevokeSystemBadge(ctx, db.RevokeSystemBadgeParams{
			TargetType: rule.targetType,
			BadgeID:    rule.badgeID,
			TargetIds:  ids,
		})

		if err != nil {
			return fmt.Errorf("%s/%s: revoking: %w", rule.badgeID, rule.targetType, err)
		}

		if awarded > 0 || revoked > 0 {
			state.Logger.Info("badge_sync: updated system badges",
				zap.String("badge", rule.badgeID),
				zap.String("targetType", rule.targetType),
				zap.Int64("awarded", awarded),
				zap.Int64("revoked", revoked))
		}
	}

	return nil
}
