package assets

import (
	"context"
	"errors"

	"popplio/db"

	"github.com/jackc/pgx/v5"
)

const (
	BenefitPremiumDays    = "premium_days"
	BenefitPriorityBoost  = "priority_boost"
	BenefitFeaturedSlot   = "featured_slot"
	BenefitSupporterBadge = "supporter_badge"
	BenefitVoteBlitz      = "vote_blitz"
	BenefitHighlight      = "highlight"
	BenefitSpotlightSlot  = "spotlight_slot"
	BenefitCardTheme      = "card_theme"
)

var recognizedBenefits = map[string]bool{
	BenefitPremiumDays:    true,
	BenefitPriorityBoost:  true,
	BenefitFeaturedSlot:   true,
	BenefitSupporterBadge: true,
	BenefitVoteBlitz:      true,
	BenefitHighlight:      true,
	BenefitSpotlightSlot:  true,
	BenefitCardTheme:      true,
}

func HasRecognizedBenefit(benefitIDs []string) bool {
	for _, id := range benefitIDs {
		if recognizedBenefits[id] {
			return true
		}
	}
	return false
}

func ApplyBenefit(ctx context.Context, tx pgx.Tx, benefitID, targetType, targetID string, durationHours int64) error {
	if targetType != "bot" && targetType != "server" {
		return errors.New("shop benefits are only supported for bots and servers today")
	}

	q := db.New(tx)
	isBot := targetType == "bot"

	switch benefitID {
	case BenefitPremiumDays:
		if isBot {
			return q.ApplyBotPremiumDays(ctx, db.ApplyBotPremiumDaysParams{Hours: int32(durationHours), BotID: targetID})
		}
		return q.ApplyServerPremiumDays(ctx, db.ApplyServerPremiumDaysParams{Hours: int32(durationHours), ServerID: targetID})
	case BenefitPriorityBoost:
		if isBot {
			return q.ApplyBotPriorityBoost(ctx, db.ApplyBotPriorityBoostParams{Hours: int32(durationHours), BotID: targetID})
		}
		return q.ApplyServerPriorityBoost(ctx, db.ApplyServerPriorityBoostParams{Hours: int32(durationHours), ServerID: targetID})
	case BenefitFeaturedSlot:
		if isBot {
			return q.ApplyBotFeaturedSlot(ctx, db.ApplyBotFeaturedSlotParams{Hours: int32(durationHours), BotID: targetID})
		}
		return q.ApplyServerFeaturedSlot(ctx, db.ApplyServerFeaturedSlotParams{Hours: int32(durationHours), ServerID: targetID})
	case BenefitSupporterBadge:
		if isBot {
			return q.ApplyBotSupporterBadge(ctx, targetID)
		}
		return q.ApplyServerSupporterBadge(ctx, targetID)
	case BenefitVoteBlitz:
		if isBot {
			return q.ApplyBotVoteBlitz(ctx, db.ApplyBotVoteBlitzParams{Hours: int32(durationHours), BotID: targetID})
		}
		return q.ApplyServerVoteBlitz(ctx, db.ApplyServerVoteBlitzParams{Hours: int32(durationHours), ServerID: targetID})
	case BenefitHighlight:
		if isBot {
			return q.ApplyBotHighlight(ctx, db.ApplyBotHighlightParams{Hours: int32(durationHours), BotID: targetID})
		}
		return q.ApplyServerHighlight(ctx, db.ApplyServerHighlightParams{Hours: int32(durationHours), ServerID: targetID})
	case BenefitSpotlightSlot:
		if isBot {
			return q.ApplyBotSpotlightSlot(ctx, db.ApplyBotSpotlightSlotParams{Hours: int32(durationHours), BotID: targetID})
		}
		return q.ApplyServerSpotlightSlot(ctx, db.ApplyServerSpotlightSlotParams{Hours: int32(durationHours), ServerID: targetID})
	case BenefitCardTheme:
		if isBot {
			return q.ApplyBotCardTheme(ctx, targetID)
		}
		return q.ApplyServerCardTheme(ctx, targetID)
	default:
		return nil
	}
}
