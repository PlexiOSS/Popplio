// Copyright (C) 2026 NodeByte LTD

package votes

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "popplio/db"
	"popplio/types"
)

func EntityGetVoteCreditsSummary(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
) (*types.VoteCreditTierRedeemSummary, error) {
	vcts, err := getVoteCreditTiers(ctx, c, targetType)

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote credit tiers [row]: %w", err)
	}

	voteCount, err := EntityGetRedeemableVoteCount(ctx, c, targetId, targetType)

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote count: %w", err)
	}

	voteInfo, err := EntityVoteInfo(ctx, c, targetId, targetType)

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote info: %w", err)
	}

	slabOverview := SlabSplitVotes(voteCount, vcts)
	totalCredits := SlabCalculateCredits(vcts, slabOverview)

	return &types.VoteCreditTierRedeemSummary{
		Tiers:        vcts,
		Votes:        voteCount,
		SlabOverview: slabOverview,
		TotalCredits: totalCredits,
		VoteInfo:     voteInfo,
	}, nil
}

func EntityRedeemVoteCredits(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
	votesToRedeem int,
) error {
	q := db.New(c)

	if err := q.LockVoteRedeemTarget(ctx, db.LockVoteRedeemTargetParams{
		TargetType: targetType,
		TargetID:   targetId,
	}); err != nil {
		return fmt.Errorf("could not acquire vote redeem lock: %w", err)
	}

	vi, err := EntityVoteInfo(ctx, c, targetId, targetType)

	if err != nil {
		return fmt.Errorf("could not fetch vote info: %w", err)
	}

	if !vi.VoteCredits {
		return errors.New("vote credits are not supported for this entity")
	}

	vcts, err := getVoteCreditTiers(ctx, c, targetType)

	if err != nil {
		return fmt.Errorf("could not fetch vote credit tiers [row]: %w", err)
	}

	voteCount, err := EntityGetRedeemableVoteCount(ctx, c, targetId, targetType)

	if err != nil {
		return fmt.Errorf("could not fetch vote count: %w", err)
	}

	if !vi.SupportsPartialVoteCreditsRedeem {
		votesToRedeem = voteCount
	}

	if votesToRedeem > voteCount {
		return errors.New("votes to redeem exceeds total votes")
	}

	slabOverview := SlabSplitVotes(votesToRedeem, vcts)
	totalCredits := SlabCalculateCredits(vcts, slabOverview)

	if totalCredits == 0 {
		return errors.New("no vote credits to redeem")
	}

	id, err := q.InsertVoteRedeemLog(ctx, db.InsertVoteRedeemLogParams{
		TargetID:   targetId,
		TargetType: targetType,
		Credits:    int32(totalCredits),
	})

	if err != nil {
		return fmt.Errorf("could not log vote credit redemption: %w", err)
	}

	if vi.SupportsPartialVoteCreditsRedeem {
		ids, err := q.GetRedeemableVoteItags(ctx, db.GetRedeemableVoteItagsParams{
			TargetID:   targetId,
			TargetType: targetType,
			Limit:      int32(votesToRedeem),
		})

		if err != nil {
			return fmt.Errorf("could not fetch vote ids: %w", err)
		}

		err = q.RedeemVotesByItags(ctx, db.RedeemVotesByItagsParams{CreditRedeem: id, Itags: ids})

		if err != nil {
			return fmt.Errorf("could not redeem vote credits: %w", err)
		}
	} else {
		err = q.RedeemAllVotesForTarget(ctx, db.RedeemAllVotesForTargetParams{
			CreditRedeem: id,
			TargetID:     targetId,
			TargetType:   targetType,
		})

		if err != nil {
			return fmt.Errorf("could not redeem vote credits: %w", err)
		}
	}

	return nil
}

func EntityGetVoteRedeemLogsSummary(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
) (*types.EntityVoteRedeemLogSummary, error) {
	rows, err := db.New(c).GetVoteRedeemLogs(ctx, db.GetVoteRedeemLogsParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote redeem logs [db fetch]: %w", err)
	}

	evrls := make([]*types.EntityVoteRedeemLog, len(rows))
	for i, row := range rows {
		var redeemedAt *time.Time
		if row.RedeemedAt.Valid {
			redeemedAt = &row.RedeemedAt.Time
		}

		evrls[i] = &types.EntityVoteRedeemLog{
			ID:              row.ID,
			TargetID:        row.TargetID,
			TargetType:      row.TargetType,
			Credits:         int(row.Credits),
			RedeemedCredits: int(row.RedeemedCredits),
			CreatedAt:       row.CreatedAt.Time,
			RedeemedAt:      redeemedAt,
		}
	}

	var totalCredits int
	var redeemedCredits int

	for i := range evrls {
		totalCredits += evrls[i].Credits
		redeemedCredits += evrls[i].RedeemedCredits
	}

	return &types.EntityVoteRedeemLogSummary{
		Redeems:          evrls,
		TotalCredits:     totalCredits,
		RedeemedCredits:  redeemedCredits,
		AvailableCredits: max(totalCredits-redeemedCredits, 0),
	}, nil
}

func getVoteCreditTiers(ctx context.Context, c DbConn, targetType string) ([]*types.VoteCreditTier, error) {
	rows, err := db.New(c).GetVoteCreditTiers(ctx, targetType)

	if err != nil {
		return nil, err
	}

	vcts := make([]*types.VoteCreditTier, len(rows))
	for i, row := range rows {
		vcts[i] = &types.VoteCreditTier{
			ID:         row.ID,
			TargetType: row.TargetType,
			Position:   int(row.Position),
			Votes:      int(row.Votes),
			Cents:      int(row.Cents),
			CreatedAt:  row.CreatedAt.Time,
		}
	}

	return vcts, nil
}

func SlabSplitVotes(votes int, tiers []*types.VoteCreditTier) []int {
	voteCredits := make([]int, len(tiers))

	var remainingVotes = votes

	for i := range tiers {
		if remainingVotes <= 0 {
			break
		}

		if remainingVotes >= tiers[i].Votes {
			voteCredits[i] = tiers[i].Votes
			remainingVotes -= tiers[i].Votes
		} else {
			voteCredits[i] = remainingVotes
			remainingVotes = 0
			break
		}
	}

	if remainingVotes > 0 {
		voteCredits[len(tiers)-1] += remainingVotes
	}

	return voteCredits
}

func SlabCalculateCredits(tiers []*types.VoteCreditTier, slab []int) int {
	var totalCredits int

	for i := range tiers {
		totalCredits += tiers[i].Cents * slab[i]
	}

	return totalCredits
}
