package cast

import (
	"context"
	"fmt"
	"strconv"

	"popplio/brandemoji"
	"popplio/db"
	"popplio/state"
	"popplio/votes"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/events"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/ptr"
	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
)

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

type Result struct {
	Votes   int
	PerUser int
	Entity  *votes.EntityInfo
}

func Cast(ctx context.Context, userID, targetType, targetID string, upvote, auto bool) (*Result, error) {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	if err := q.LockVoteCastTarget(ctx, db.LockVoteCastTargetParams{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
	}); err != nil {
		return nil, fmt.Errorf("acquiring vote lock: %w", err)
	}

	entityInfo, err := votes.GetEntityInfo(ctx, tx, targetID, targetType)

	if err != nil {
		state.Logger.Error("Failed to fetch entity info", zap.Error(err), zap.String("userId", userID), zap.String("targetId", targetID), zap.String("targetType", targetType))
		return nil, &Error{Message: "Failed to fetch entity info."}
	}

	vi, err := votes.EntityVoteCheck(ctx, tx, userID, targetID, targetType)

	if err != nil {
		return nil, fmt.Errorf("checking vote: %w", err)
	}

	if !vi.VoteInfo.SupportsDownvotes && !upvote {
		return nil, &Error{Message: "This entity does not support downvotes"}
	}

	if !vi.VoteInfo.SupportsUpvotes && upvote {
		return nil, &Error{Message: "This entity does not support upvotes"}
	}

	if vi.HasVoted {
		if !vi.VoteInfo.MultipleVotes {
			if vi.ValidVotes[0].Upvote == upvote {
				return nil, &Error{Message: "You have already voted for this entity before!"}
			}

			if err := q.DeleteUserEntityVotes(ctx, db.DeleteUserEntityVotesParams{
				Author:     userID,
				TargetID:   targetID,
				TargetType: targetType,
			}); err != nil {
				return nil, fmt.Errorf("deleting old vote: %w", err)
			}
		} else {
			timeStr := "a while"

			if vi.Wait != nil {
				timeStr = fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", vi.Wait.Hours, vi.Wait.Minutes, vi.Wait.Seconds)
			}

			if len(vi.ValidVotes) > 1 {
				return nil, &Error{Message: "Please wait " + timeStr + " before voting again. Your last vote was also a double vote!"}
			}

			return nil, &Error{Message: "Please wait " + timeStr + " before voting again"}
		}
	}

	if err := votes.EntityGiveVotes(ctx, tx, upvote, auto, userID, targetType, targetID, vi.VoteInfo); err != nil {
		return nil, fmt.Errorf("giving votes: %w", err)
	}

	if err := votes.EntityPostVote(ctx, tx, targetType, targetID); err != nil {
		return nil, fmt.Errorf("post-vote tasks: %w", err)
	}

	nvc, err := votes.EntityGetVoteCount(ctx, tx, targetID, targetType)

	if err != nil {
		return nil, fmt.Errorf("fetching new vote count: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing vote: %w", err)
	}

	return &Result{Votes: nvc, PerUser: vi.VoteInfo.PerUser, Entity: entityInfo}, nil
}

func Announce(userID, targetType, targetID string, res *Result, auto bool) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while sending vote log message", zap.Any("panic", rec), zap.String("userId", userID), zap.String("targetId", targetID), zap.String("targetType", targetType))
			}
		}()

		userObj, err := dovewing.GetUser(state.Context, userID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch user info", zap.Error(err), zap.String("userId", userID), zap.String("targetId", targetID), zap.String("targetType", targetType))
			return
		}

		description := brandemoji.Popplio.Get("upvote", "❤️") + " " + userObj.DisplayName + " has voted for " + targetType + ": " + res.Entity.Name

		if auto {
			description += " (auto-vote)"
		}

		_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.VoteLogs, discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					URL: res.Entity.URL,
					Thumbnail: &discord.EmbedResource{
						URL: res.Entity.Avatar,
					},
					Title:       "🎉 Vote Count Updated!",
					Description: description,
					Color:       0x8A6BFD,
					Fields: []discord.EmbedField{
						{
							Name:   "Vote Count:",
							Value:  strconv.Itoa(res.Votes),
							Inline: ptr.TruePtr,
						},
						{
							Name:   "Votes Added:",
							Value:  strconv.Itoa(res.PerUser),
							Inline: ptr.TruePtr,
						},
						{
							Name:   "User ID:",
							Value:  userObj.ID,
							Inline: ptr.TruePtr,
						},
						{
							Name:   "View " + targetType + "'s page",
							Value:  "[View " + res.Entity.Name + "](" + res.Entity.URL + ")",
							Inline: ptr.TruePtr,
						},
					},
				},
			},
		})

		if err != nil {
			state.Logger.Error("Failed to send vote log message", zap.Error(err), zap.String("userId", userID), zap.String("targetId", targetID), zap.String("targetType", targetType))
		}
	}()

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while sending vote webhook", zap.Any("panic", rec), zap.String("userId", userID), zap.String("targetId", targetID), zap.String("targetType", targetType))
			}
		}()

		err := drivers.Send(drivers.With{
			UserID:     userID,
			TargetID:   targetID,
			TargetType: targetType,
			Data: events.WebhookNewVoteData{
				Votes:   res.Votes,
				PerUser: res.PerUser,
				Auto:    auto,
			},
		})

		if err != nil {
			state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("userId", userID), zap.String("targetId", targetID), zap.String("targetType", targetType))
		}
	}()
}
