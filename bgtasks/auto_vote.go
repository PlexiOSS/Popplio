package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"popplio/captcha"
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

	var cast_, failed int

	for _, t := range targets {
		if !inSchedule(t.Days, int(t.StartMinute), int(t.EndMinute), t.Timezone, time.Now()) {
			continue
		}

		voted, reason, err := autoVoteOne(ctx, q, t.UserID, t.TargetType, t.TargetID)

		if err != nil {
			failed++
			state.Logger.Error("Auto-vote failed", zap.Error(err), zap.String("userId", t.UserID), zap.String("targetType", t.TargetType), zap.String("targetId", t.TargetID))
			continue
		}

		if voted {
			cast_++
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

	needsCaptcha, err := captcha.RequiresCaptcha(ctx, targetType, targetID)

	if err != nil {
		return false, "", err
	}

	if needsCaptcha {
		return false, "This listing requires a captcha to vote, so it has to be voted for by hand.", nil
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

func notifyAutoVoteSkipped(ctx context.Context, userID, targetType, targetID, reason string) {
	name := "a listing"

	if info, err := votes.GetEntityInfo(ctx, state.Pool, targetID, targetType); err == nil {
		name = info.Name
	}

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

func inSchedule(days []int16, startMinute, endMinute int, timezone string, now time.Time) bool {
	loc, err := time.LoadLocation(timezone)

	if err != nil {
		loc = time.UTC
	}

	local := now.In(loc)

	if len(days) > 0 && !slices.Contains(days, int16(local.Weekday())) {
		return false
	}

	minute := local.Hour()*60 + local.Minute()

	return minute >= startMinute && minute < endMinute
}
