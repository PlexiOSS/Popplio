package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"popplio/audit"
	"popplio/autovote"
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/votes"
	"popplio/votes/cast"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func AutoVote(ctx context.Context) error {
	q := db.New(state.Pool)

	targets, err := q.ListDueAutoVoteTargets(ctx)

	if err != nil {
		return fmt.Errorf("listing auto-vote targets: %w", err)
	}

	overCardLimit := map[string]bool{}

	if ids, err := q.ListAutoVoteUsersOverCardLimit(ctx, autovote.MaxAccountsPerCard); err != nil {
		return fmt.Errorf("checking auto-vote card limits: %w", err)
	} else {
		for _, id := range ids {
			overCardLimit[id] = true
		}
	}

	var cast_, failed int

	for _, t := range targets {
		if !autovote.InSchedule(t.Days, int(t.StartMinute), int(t.EndMinute), t.Timezone, time.Now()) {
			continue
		}

		var (
			voted  bool
			reason string
			err    error
		)

		if overCardLimit[t.UserID] {
			reason = autovote.CardLimitSkipReason
		} else {
			voted, reason, err = autoVoteOne(ctx, q, t.UserID, t.TargetType, t.TargetID)
		}

		if err != nil {
			failed++
			audit.Record(audit.Event{Source: audit.SourceSystem, ActorType: audit.ActorUser, ActorID: t.UserID, Action: "autovote.failed", TargetType: t.TargetType, TargetID: t.TargetID, Data: map[string]any{"error": err.Error()}})
			state.Logger.Error("Auto-vote failed", zap.Error(err), zap.String("userId", t.UserID), zap.String("targetType", t.TargetType), zap.String("targetId", t.TargetID))
			continue
		}

		if voted {
			cast_++
			audit.Record(audit.Event{Source: audit.SourceSystem, ActorType: audit.ActorUser, ActorID: t.UserID, Action: "autovote.voted", TargetType: t.TargetType, TargetID: t.TargetID})

			if t.Notify == "each" {
				notifyAutoVoted(ctx, t.UserID, t.TargetType, t.TargetID)
			}

			continue
		}

		if reason != "" {
			if err := q.MarkAutoVoteTargetSkipped(ctx, db.MarkAutoVoteTargetSkippedParams{
				UserID:         t.UserID,
				TargetType:     t.TargetType,
				TargetID:       t.TargetID,
				LastSkipReason: pgtype.Text{String: reason, Valid: true},
			}); err != nil {
				state.Logger.Warn("Failed to record auto-vote skip", zap.Error(err), zap.String("userId", t.UserID))
			}

			if !t.LastSkipReason.Valid || t.LastSkipReason.String != reason {
				audit.Record(audit.Event{Source: audit.SourceSystem, ActorType: audit.ActorUser, ActorID: t.UserID, Action: "autovote.skipped", TargetType: t.TargetType, TargetID: t.TargetID, Data: map[string]any{"reason": reason}})
				notifyAutoVoteSkipped(ctx, t.UserID, t.TargetType, t.TargetID, reason)
			}
		}
	}

	if cast_ > 0 {
		state.Logger.Info("Auto-votes cast", zap.Int("count", cast_))
	}

	if failed > 0 {
		return fmt.Errorf("%d auto-votes failed", failed)
	}

	return nil
}

func autoVoteOne(ctx context.Context, q *db.Queries, userID, targetType, targetID string) (bool, string, error) {
	check, err := votes.EntityVoteCheck(ctx, state.Pool, userID, targetID, targetType)

	if err != nil {
		return false, "", err
	}

	if check.HasVoted {
		return false, "", nil
	}

	var optOut bool

	switch targetType {
	case "bot":
		optOut, err = q.GetBotAutoVotesOptOut(ctx, targetID)
	case "server":
		optOut, err = q.GetServerAutoVotesOptOut(ctx, targetID)
	default:
		return false, "This kind of listing can't be auto-voted for.", nil
	}

	if err != nil {
		return false, "This listing no longer exists.", nil
	}

	if optOut {
		return false, "The owner has turned off automatic votes for this listing.", nil
	}

	banned, err := q.GetUserVoteBanned(ctx, userID)

	if err != nil {
		return false, "", err
	}

	if banned {
		return false, "You're currently banned from voting.", nil
	}

	res, err := cast.Cast(ctx, userID, targetType, targetID, true, true)

	if err != nil {
		var castErr *cast.Error

		if errors.As(err, &castErr) {
			return false, castErr.Message, nil
		}

		return false, "", err
	}

	if err := q.MarkAutoVoteTargetVoted(ctx, db.MarkAutoVoteTargetVotedParams{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
	}); err != nil {
		state.Logger.Warn("Failed to record auto-vote", zap.Error(err), zap.String("userId", userID))
	}

	cast.Announce(userID, targetType, targetID, res, true)

	return true, "", nil
}

func listingName(ctx context.Context, targetType, targetID string) string {
	if info, err := votes.GetEntityInfo(ctx, state.Pool, targetID, targetType); err == nil {
		return info.Name
	}

	return "a listing"
}

func notifyAutoVoted(ctx context.Context, userID, targetType, targetID string) {
	name := listingName(ctx, targetType, targetID)

	if err := notifications.PushNotification(userID, types.Alert{
		Type:     types.AlertTypeSuccess,
		Title:    "Auto-voted for " + name,
		Message:  "Auto-vote cast your vote for " + name + ".",
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/dashboard?tab=auto-vote", Valid: true},
		Category: types.AlertCategoryVotes,
	}); err != nil {
		state.Logger.Warn("Failed to send auto-vote alert", zap.Error(err), zap.String("userId", userID))
	}
}

func AutoVoteSummary(ctx context.Context) error {
	q := db.New(state.Pool)

	due, err := q.ListDueAutoVoteSummaries(ctx)

	if err != nil {
		return fmt.Errorf("listing auto-vote summaries: %w", err)
	}

	for _, u := range due {
		since := time.Now().Add(-24 * time.Hour)

		if u.LastSummaryAt.Valid {
			since = u.LastSummaryAt.Time
		}

		counts, err := q.CountUserAutoVotesSince(ctx, db.CountUserAutoVotesSinceParams{
			Author: u.UserID,
			Since:  pgtype.Timestamp{Time: since.UTC(), Valid: true},
		})

		if err != nil {
			state.Logger.Warn("Failed to count auto-votes for a summary", zap.Error(err), zap.String("userId", u.UserID))
			continue
		}

		if len(counts) > 0 {
			total := 0
			parts := make([]string, 0, len(counts))

			for _, c := range counts {
				total += int(c.Votes)
				parts = append(parts, fmt.Sprintf("%s (%d)", listingName(ctx, c.TargetType, c.TargetID), c.Votes))
			}

			plural := "s"

			if total == 1 {
				plural = ""
			}

			if err := notifications.PushNotification(u.UserID, types.Alert{
				Type:     types.AlertTypeInfo,
				Title:    fmt.Sprintf("Auto-vote cast %d vote%s today", total, plural),
				Message:  "In the last day: " + strings.Join(parts, ", ") + ".",
				URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/dashboard?tab=auto-vote", Valid: true},
				Category: types.AlertCategoryVotes,
			}); err != nil {
				state.Logger.Warn("Failed to send auto-vote summary", zap.Error(err), zap.String("userId", u.UserID))
				continue
			}
		}

		if err := q.MarkAutoVoteSummarySent(ctx, u.UserID); err != nil {
			state.Logger.Warn("Failed to record an auto-vote summary", zap.Error(err), zap.String("userId", u.UserID))
		}
	}

	return nil
}

func notifyAutoVoteSkipped(ctx context.Context, userID, targetType, targetID, reason string) {
	name := listingName(ctx, targetType, targetID)

	if err := notifications.PushNotification(userID, types.Alert{
		Type:     types.AlertTypeWarning,
		Title:    "Auto-vote skipped " + name,
		Message:  reason + " Auto-vote will keep trying, or you can swap it for another listing.",
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/dashboard?tab=auto-vote", Valid: true},
		Category: types.AlertCategoryVotes,
	}); err != nil {
		state.Logger.Warn("Failed to send auto-vote skipped alert", zap.Error(err), zap.String("userId", userID))
	}
}
