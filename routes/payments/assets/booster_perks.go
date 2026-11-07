// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"time"

	"popplio/boosters"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const RenewalDays = 30

func LiveBoosting(userID string) (inServer, boosting bool) {
	id, err := snowflake.Parse(userID)

	if err != nil {
		return false, false
	}

	return boosters.Check(id)
}

func BoosterPerksFor(ctx context.Context, userID string) (types.BoosterPerks, error) {
	q := db.New(state.Pool)
	inServer, boosting := LiveBoosting(userID)

	out := types.BoosterPerks{
		Boosting:      boosting,
		InServer:      inServer,
		PremiumDays:   boosterHours() / 24,
		AutoVoteSlots: 1,
		RenewalDays:   RenewalDays,
	}

	row, err := q.GetBoosterPerks(ctx, userID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}

	if err == nil {
		if row.BoostingSince.Valid && row.Boosting {
			since := row.BoostingSince.Time
			out.BoostingSince = &since
		}

		if row.LastError.Valid {
			out.LastError = &row.LastError.String
		}

		if row.LastBoosterClaim.Valid {
			at := row.LastBoosterClaim.Time
			out.LastClaimAt = &at
		}

		if row.TargetType.Valid && row.TargetID.Valid {
			target := &types.BoosterPerkTarget{TargetType: row.TargetType.String, TargetID: row.TargetID.String}

			if info, err := votes.GetEntityInfo(ctx, state.Pool, target.TargetID, target.TargetType); err == nil {
				target.Name = info.Name
				target.Avatar = info.Avatar
			}

			out.Target = target
		}
	} else if claim, err := q.GetUserLastBoosterClaim(ctx, userID); err == nil && claim.Valid {
		at := claim.Time
		out.LastClaimAt = &at
	}

	if out.Boosting && out.Target != nil {
		next := time.Now()

		if out.LastClaimAt != nil && out.LastClaimAt.Add(RenewalDays*24*time.Hour).After(next) {
			next = out.LastClaimAt.Add(RenewalDays * 24 * time.Hour)
		}

		out.NextRenewalAt = &next
	}

	return out, nil
}

const BoosterPlan = "bronze"

type TargetError struct {
	Reason string
}

func (e *TargetError) Error() string {
	return e.Reason
}

func boosterHours() int {
	for _, p := range Plans {
		if p.ID == BoosterPlan {
			return p.TimePeriod
		}
	}

	return 24 * 30
}

func ApplyBoosterPremium(ctx context.Context, userID, targetType, targetID string, now time.Time) (int, error) {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return 0, err
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	var listingType string

	switch targetType {
	case "bot":
		row, err := q.GetBotTypePremium(ctx, targetID)
		listingType = row.Type

		if errors.Is(err, pgx.ErrNoRows) {
			return 0, &TargetError{"the bot no longer exists"}
		} else if err != nil {
			return 0, err
		}
	case "server":
		row, err := q.GetServerTypePremium(ctx, targetID)
		listingType = row.Type

		if errors.Is(err, pgx.ErrNoRows) {
			return 0, &TargetError{"the server no longer exists"}
		} else if err != nil {
			return 0, err
		}
	default:
		return 0, &TargetError{"that listing type can't get premium"}
	}

	if listingType != "approved" && listingType != "certified" {
		return 0, &TargetError{"the " + targetType + " isn't approved"}
	}

	hours := boosterHours()

	inserted, err := q.InsertPaymentFulfillment(ctx, db.InsertPaymentFulfillmentParams{
		Provider:     "booster",
		ExternalID:   "booster:" + userID + ":" + now.UTC().Format("2006-01-02"),
		UserID:       userID,
		ProductID:    "premium",
		ForType:      targetType,
		ForID:        targetID,
		PremiumHours: pgtype.Int4{Int32: int32(hours), Valid: true},
	})

	if err != nil {
		return 0, err
	}

	if inserted == 0 {
		return 0, nil
	}

	if targetType == "bot" {
		err = q.ApplyBotPremiumDays(ctx, db.ApplyBotPremiumDaysParams{Hours: int32(hours), BotID: targetID})
	} else {
		err = q.ApplyServerPremiumDays(ctx, db.ApplyServerPremiumDaysParams{Hours: int32(hours), ServerID: targetID})
	}

	if err != nil {
		return 0, err
	}

	if err := q.UpdateUserLastBoosterClaim(ctx, userID); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return hours, nil
}
