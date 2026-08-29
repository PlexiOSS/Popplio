package votereminders

import (
	"fmt"
	"time"

	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func VrLoop() {
	for {
		err := vrCheck()

		if err != nil {
			state.Logger.Error("vrCheck returned an error", zap.Error(err))
			time.Sleep(5 * time.Minute)
			continue
		}

		time.Sleep(10 * time.Second)
	}
}

func vrCheck() error {
	q := db.New(state.Pool)

	rows, err := q.GetStaleUserReminders(state.Context)

	if err != nil {
		return fmt.Errorf("error finding reminders: %w", err)
	}

	for _, row := range rows {
		userId, targetId, targetType := row.UserID, row.TargetID, row.TargetType

		vi, err := votes.EntityVoteCheck(state.Context, state.Pool, userId, targetId, targetType)

		if err != nil {
			state.Logger.Error("Error checking votes of entity", zap.Error(err), zap.String("userId", userId), zap.String("targetId", targetId), zap.String("targetType", targetType))
			continue
		}

		if !vi.HasVoted {
			entityInfo, err := votes.GetEntityInfo(state.Context, state.Pool, targetId, targetType)

			if err != nil {
				state.Logger.Error("Error finding bot info", zap.Error(err), zap.String("targetId", targetId), zap.String("targetType", targetType))
				continue
			}

			message := types.Alert{
				Type:     types.AlertTypeInfo,
				URL:      pgtype.Text{String: entityInfo.VoteURL, Valid: true},
				Message:  "You can vote for the " + targetType + " " + entityInfo.Name + " now!",
				Title:    "Vote for " + entityInfo.Name + "!",
				Icon:     entityInfo.Avatar,
				Category: types.AlertCategoryVotes,
			}

			err = notifications.PushNotification(userId, message)

			if err != nil {
				state.Logger.Error("PushNotification returned an error", zap.Error(err), zap.String("userId", userId), zap.String("targetId", targetId), zap.String("targetType", targetType))
				continue
			}

			err = q.TouchUserReminder(state.Context, db.TouchUserReminderParams{
				UserID:     userId,
				TargetID:   targetId,
				TargetType: targetType,
			})
			if err != nil {
				state.Logger.Error("Error updating user reminder", zap.Error(err), zap.String("userId", userId), zap.String("targetId", targetId), zap.String("targetType", targetType))
				continue
			}

		}
	}

	return nil
}
