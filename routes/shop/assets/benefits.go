// Package assets implements what buying a shop item actually does.
//
// shop_items/shop_item_benefits are a staff-managed catalogue (full CRUD via
// the Arcadia panel) with no field describing what a benefit's effect is —
// Name/Description are free text for display only. The IDs below are the
// bridge: staff must give a shop_item_benefits row one of these exact IDs
// for a purchase to actually do anything. Unrecognized benefit IDs are
// silently skipped so staff can still catalogue purely descriptive/future
// benefits without breaking a purchase.
package assets

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const (
	BenefitPremiumDays    = "premium_days"
	BenefitPriorityBoost  = "priority_boost"
	BenefitFeaturedSlot   = "featured_slot"
	BenefitSupporterBadge = "supporter_badge"
	BenefitVoteBlitz      = "vote_blitz"
)

var recognizedBenefits = map[string]bool{
	BenefitPremiumDays:    true,
	BenefitPriorityBoost:  true,
	BenefitFeaturedSlot:   true,
	BenefitSupporterBadge: true,
	BenefitVoteBlitz:      true,
}

// HasRecognizedBenefit reports whether at least one of the given benefit IDs
// has a defined effect — used to reject purchasing an item that would
// otherwise just spend credits for nothing.
func HasRecognizedBenefit(benefitIDs []string) bool {
	for _, id := range benefitIDs {
		if recognizedBenefits[id] {
			return true
		}
	}
	return false
}

// ApplyBenefit applies one recognized shop benefit's effect to a target
// entity (bot or server — both tables carry identical columns for every
// benefit below). durationHours comes from the purchased ShopItem's
// Duration field.
func ApplyBenefit(ctx context.Context, tx pgx.Tx, benefitID, targetType, targetID string, durationHours int64) error {
	var table, idCol string

	switch targetType {
	case "bot":
		table, idCol = "bots", "bot_id"
	case "server":
		table, idCol = "servers", "server_id"
	default:
		return errors.New("shop benefits are only supported for bots and servers today")
	}

	switch benefitID {
	case BenefitPremiumDays:
		// Matches routes/payments/assets/give_perks.go's GivePerks exactly, so
		// a credits-bought premium period behaves identically to a paid one.
		_, err := tx.Exec(ctx,
			"UPDATE "+table+" SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE "+idCol+" = $2",
			durationHours, targetID,
		)
		return err
	case BenefitPriorityBoost:
		_, err := tx.Exec(ctx,
			"UPDATE "+table+" SET boosted_until = GREATEST(COALESCE(boosted_until, NOW()), NOW()) + make_interval(hours => $1) WHERE "+idCol+" = $2",
			durationHours, targetID,
		)
		return err
	case BenefitFeaturedSlot:
		_, err := tx.Exec(ctx,
			"UPDATE "+table+" SET featured_until = GREATEST(COALESCE(featured_until, NOW()), NOW()) + make_interval(hours => $1) WHERE "+idCol+" = $2",
			durationHours, targetID,
		)
		return err
	case BenefitSupporterBadge:
		_, err := tx.Exec(ctx, "UPDATE "+table+" SET supporter_badge = true WHERE "+idCol+" = $1", targetID)
		return err
	case BenefitVoteBlitz:
		_, err := tx.Exec(ctx,
			"UPDATE "+table+" SET vote_blitz_until = GREATEST(COALESCE(vote_blitz_until, NOW()), NOW()) + make_interval(hours => $1) WHERE "+idCol+" = $2",
			durationHours, targetID,
		)
		return err
	default:
		return nil // unrecognized benefit — no-op, not an error
	}
}
