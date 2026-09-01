package votes

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "popplio/db"
	"popplio/types"
)

// Returns a summary of the vote credit tiers of an entity
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

	// Redeemable, not the entity's public total -- a vote already cashed in
	// by an earlier redemption must not be offered (or paid out) again.
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

// Redeems vote credits for a user towards a specific entity
func EntityRedeemVoteCredits(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
	votesToRedeem int,
) error {
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

	// Redeemable, not the entity's public total -- see EntityGetVoteCreditsSummary.
	voteCount, err := EntityGetRedeemableVoteCount(ctx, c, targetId, targetType)

	if err != nil {
		return fmt.Errorf("could not fetch vote count: %w", err)
	}

	if !vi.SupportsPartialVoteCreditsRedeem {
		votesToRedeem = voteCount // If the entity does not support partial vote credit redemption, then redeem all votes
	}

	// Check if the votes to redeem exceeds the total votes to protect against malicious input for votes to redeem
	if votesToRedeem > voteCount {
		return errors.New("votes to redeem exceeds total votes")
	}

	slabOverview := SlabSplitVotes(votesToRedeem, vcts)
	totalCredits := SlabCalculateCredits(vcts, slabOverview)

	if totalCredits == 0 {
		return errors.New("no vote credits to redeem")
	}

	q := db.New(c)

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

// Returns a summary of the entity vote redeem logs
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

// getVoteCreditTiers fetches and converts the vote credit tiers for a
// target type, shared by EntityGetVoteCreditsSummary and
// EntityRedeemVoteCredits.
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

// Given a number of votes and the vote credit tiers, return the structure of how vote credits should be awarded
// as a map of string to int
//
// Note that this function assumes that the vote credits tiers are sorted by position in ascending order
func SlabSplitVotes(votes int, tiers []*types.VoteCreditTier) []int {
	/*
		<div class="system">
				<p>
					Vote credits are tier based through slabs<br /><br />

					(e.g.)For the following tiers<br /><br />
				</p>
				<OrderedList>
					<ListItem>Tier 1: 100 votes at 0.10 cents</ListItem>
					<ListItem>Tier 2: 200 votes at 0.05 cents</ListItem>
					<ListItem>Tier 3: 50 votes at 0.025 cents</ListItem>
				</OrderedList>
				<p>Would mean 625 votes would be split as the following:</p>
				<OrderedList>
					<ListItem>100 votes: 0.10 cents [Tier 1]</ListItem>
					<ListItem>Next 200 votes: 0.05 cents [Tier 2]</ListItem>
					<ListItem>Next 50 votes: 0.025 cents [Tier 3]</ListItem>
					<ListItem>Last 275 votes: 0.025 cents [last tier used at end of tiering]</ListItem>
				</OrderedList>
			</div>
	*/

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

	// If there are remaining votes, then add them to the last tier
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
