// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"

	"popplio/autovote"
	"popplio/db"
	"popplio/state"

	"go.uber.org/zap"
)

const AutoVoteReporterID = "system:autovote"

func AutoVoteWatch(ctx context.Context) error {
	q := db.New(state.Pool)

	rows, err := q.ListAutoVoteShareSpikes(ctx, autovote.SpikeMinAutoVotes)

	if err != nil {
		return fmt.Errorf("listing auto-vote shares: %w", err)
	}

	filed := 0

	for _, r := range rows {
		if !autovote.IsShareSpike(int(r.AutoDay), int(r.TotalDay), int(r.AutoBefore), int(r.TotalBefore)) {
			continue
		}

		before := 0

		if r.TotalBefore > 0 {
			before = int(r.AutoBefore) * 100 / int(r.TotalBefore)
		}

		description := fmt.Sprintf(
			"Automatically filed: %d of the %d upvotes in the last 24 hours (%d%%) were automatic, from %d Auto-vote accounts, up from %d%% over the 7 days before. Check whether Auto-vote subscriptions were bought to boost this listing.",
			r.AutoDay, r.TotalDay, int(r.AutoDay)*100/int(r.TotalDay), r.AutoVoters, before,
		)

		n, err := q.InsertSystemReport(ctx, db.InsertSystemReportParams{
			TargetType:  r.TargetType,
			TargetID:    r.TargetID,
			ReporterID:  AutoVoteReporterID,
			Description: description,
		})

		if err != nil {
			state.Logger.Warn("auto_vote_watch: failed to file report", zap.Error(err), zap.String("targetType", r.TargetType), zap.String("targetId", r.TargetID))
			continue
		}

		filed += int(n)
	}

	if filed > 0 {
		state.Logger.Info("auto_vote_watch: filed reports", zap.Int("count", filed))
	}

	return nil
}
