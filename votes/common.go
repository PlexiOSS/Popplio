package votes

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/PlexiOSS/Keel/dovewing"
)

type DbConn = db.DBTX

func GetDoubleVote() bool {
	weekday := time.Now().UTC().Weekday()
	return weekday == time.Friday || weekday == time.Saturday || weekday == time.Sunday
}

type EntityInfo struct {
	Name    string
	URL     string
	VoteURL string
	Avatar  string
}

func GetEntityInfo(ctx context.Context, c DbConn, targetId, targetType string) (*EntityInfo, error) {
	q := db.New(c)

	switch targetType {
	case "bot":
		row, err := q.GetBotVoteStatus(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bot not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch bot data for this vote: %w", err)
		}

		if row.VoteBanned {
			return nil, errors.New("bot is vote banned and cannot be voted for right now")
		}

		if row.Type != "approved" && row.Type != "certified" {
			return nil, errors.New("bot is not approved or certified and cannot be voted for right now")
		}

		botObj, err := dovewing.GetUser(ctx, targetId, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, err
		}

		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/bots/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/bots/" + targetId,
			Name:    botObj.Username,
			Avatar:  botObj.Avatar,
		}, nil
	case "pack":
		voteBanned, err := q.GetPackVoteBanned(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("pack not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch pack data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("pack is vote banned and cannot be voted for right now")
		}

		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/packs/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/packs/" + targetId,
			Name:    targetId,
		}, nil
	case "team":
		row, err := q.GetTeamVoteStatus(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("team not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch team data for this vote: %w", err)
		}

		if row.VoteBanned {
			return nil, errors.New("team is vote banned and cannot be voted for right now")
		}

		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/teams/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/teams/" + targetId,
			Name:    row.Name,
		}, nil
	case "server":
		row, err := q.GetServerVoteStatus(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch server data for this vote: %w", err)
		}

		if row.VoteBanned {
			return nil, errors.New("server is vote banned and cannot be voted for right now")
		}

		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/servers/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/servers/" + targetId,
			Name:    row.Name,
		}, nil
	case "blog":
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/blog/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/blog/" + targetId,
			Name:    targetId,
		}, nil
	default:
		return nil, errors.New("unimplemented target type:" + targetType)
	}
}

func EntityVoteInfo(ctx context.Context, c DbConn, targetId, targetType string) (*types.VoteInfo, error) {
	var voteEntity = types.VoteInfo{
		PerUser:           1,
		VoteTime:          12,
		MultipleVotes:     true,
		VoteCredits:       false,
		SupportsUpvotes:   true,
		SupportsDownvotes: true,
	}

	q := db.New(c)

	switch targetType {
	case "bot":
		voteEntity.VoteCredits = true

		row, err := q.GetBotVoteInfo(ctx, targetId)

		if err != nil {
			return nil, err
		}

		premium, voteBlitzUntil := row.Premium, row.VoteBlitzUntil

		if premium {
			voteEntity.VoteTime = 4
		} else {
			if GetDoubleVote() {
				voteEntity.PerUser = 2
				voteEntity.VoteTime = 6
				voteEntity.WeekendBonus = true
			}
		}

		if voteBlitzUntil.Valid && voteBlitzUntil.Time.After(time.Now()) {
			voteEntity.VoteTime = max(voteEntity.VoteTime/2, 1)
		}
	case "server":
		voteEntity.VoteCredits = true

		row, err := q.GetServerVoteInfo(ctx, targetId)

		if err != nil {
			return nil, err
		}

		premium, voteBlitzUntil := row.Premium, row.VoteBlitzUntil

		if premium {
			voteEntity.VoteTime = 4
		} else {
			if GetDoubleVote() {
				voteEntity.PerUser = 2
				voteEntity.VoteTime = 6
				voteEntity.WeekendBonus = true
			}
		}

		if voteBlitzUntil.Valid && voteBlitzUntil.Time.After(time.Now()) {
			voteEntity.VoteTime = max(voteEntity.VoteTime/2, 1)
		}
	case "blog":
		voteEntity.MultipleVotes = false
		voteEntity.PerUser = 1
	case "team":
		voteEntity.VoteCredits = true

		if GetDoubleVote() {
			voteEntity.PerUser = 2
			voteEntity.VoteTime = 6
			voteEntity.WeekendBonus = true
		}
	case "pack":
		voteEntity.VoteCredits = true

		if GetDoubleVote() {
			voteEntity.WeekendBonus = true
			voteEntity.PerUser = 2
			voteEntity.VoteTime = 6
		}
	}

	return &voteEntity, nil
}

func EntityVoteCheck(ctx context.Context, c DbConn, userId, targetId, targetType string) (*types.UserVote, error) {
	vi, err := EntityVoteInfo(ctx, c, targetId, targetType)

	if err != nil {
		return nil, err
	}

	rows, err := db.New(c).GetEntityVotes(ctx, db.GetEntityVotesParams{
		Author:     userId,
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return nil, err
	}

	validVotes := make([]types.EntityVote, len(rows))
	for i, row := range rows {
		validVotes[i] = types.EntityVote{
			ITag:       row.Itag,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			AuthorID:   row.Author,
			Upvote:     row.Upvote,
			Void:       row.Void,
			VoidReason: row.VoidReason,
			VoidedAt:   pgtype.Timestamp{Time: row.VoidedAt.Time, Valid: row.VoidedAt.Valid},
			CreatedAt:  row.CreatedAt.Time,
			VoteNum:    int(row.VoteNum),
			Credit:     row.CreditRedeem,
			Immutable:  row.Immutable,
		}
	}

	var vw *types.VoteWait

	var hasVoted bool

	if vi.MultipleVotes {
		if len(validVotes) > 0 {
			hasVoted = validVotes[0].CreatedAt.Add(time.Duration(vi.VoteTime) * time.Hour).After(time.Now())

			if hasVoted {
				timeElapsed := time.Since(validVotes[0].CreatedAt)

				timeToWait := int64(vi.VoteTime)*60*60*1000 - timeElapsed.Milliseconds()

				timeToWaitTime := (time.Duration(timeToWait) * time.Millisecond)

				hours := timeToWaitTime / time.Hour
				mins := (timeToWaitTime - (hours * time.Hour)) / time.Minute
				secs := (timeToWaitTime - (hours*time.Hour + mins*time.Minute)) / time.Second

				vw = &types.VoteWait{
					Hours:   int(hours),
					Minutes: int(mins),
					Seconds: int(secs),
				}
			}
		}
	} else {
		hasVoted = len(validVotes) > 0
	}

	return &types.UserVote{
		HasVoted:   hasVoted,
		ValidVotes: validVotes,
		VoteInfo:   vi,
		Wait:       vw,
	}, nil
}

func EntityGetVoteCount(ctx context.Context, c DbConn, targetId, targetType string) (int, error) {
	row, err := db.New(c).CountEntityVotes(ctx, db.CountEntityVotesParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return 0, err
	}

	return int(row.Upvotes - row.Downvotes), nil
}

// EntityGetRedeemableVoteCount is EntityGetVoteCount's counterpart for the
// vote-credit redemption flow specifically: it excludes votes an earlier
// redemption already claimed (credit_redeem IS NOT NULL), so a repeat
// redemption can't pay out credits for the same vote twice. The public vote
// count (EntityGetVoteCount, used everywhere else -- listings, profiles,
// leaderboards) intentionally keeps counting a redeemed vote; only the
// credit math itself needs the narrower count.
func EntityGetRedeemableVoteCount(ctx context.Context, c DbConn, targetId, targetType string) (int, error) {
	row, err := db.New(c).CountRedeemableEntityVotes(ctx, db.CountRedeemableEntityVotesParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return 0, err
	}

	return int(row.Upvotes - row.Downvotes), nil
}

func EntityGiveVotes(ctx context.Context, c DbConn, upvote bool, author, targetType, targetId string, vi *types.VoteInfo) error {
	q := db.New(c)

	for i := 0; i < vi.PerUser; i++ {
		err := q.InsertEntityVote(ctx, db.InsertEntityVoteParams{
			Author:     author,
			TargetID:   targetId,
			TargetType: targetType,
			Upvote:     upvote,
			VoteNum:    int32(i),
		})

		if err != nil {
			return fmt.Errorf("failed to insert vote: %w", err)
		}
	}
	return nil
}

func EntityPostVote(ctx context.Context, c DbConn, targetType, targetId string) error {
	nvc, err := EntityGetVoteCount(ctx, c, targetId, targetType)

	if err != nil {
		return fmt.Errorf("failed to get vote count: %w", err)
	}

	q := db.New(c)

	switch targetType {
	case "bot":
		err = q.UpdateBotApproximateVotes(ctx, db.UpdateBotApproximateVotesParams{ApproximateVotes: int32(nvc), BotID: targetId})
	case "server":
		err = q.UpdateServerApproximateVotes(ctx, db.UpdateServerApproximateVotesParams{ApproximateVotes: int32(nvc), ServerID: targetId})
	case "team":
		err = q.UpdateTeamApproximateVotes(ctx, db.UpdateTeamApproximateVotesParams{ApproximateVotes: int32(nvc), ID: targetId})
	}

	if err != nil {
		return fmt.Errorf("failed to update vote count: %w", err)
	}

	return nil
}

func RecomputeApproximateVotes(ctx context.Context, c DbConn, targetType string) error {
	q := db.New(c)

	switch targetType {
	case "bot":
		if err := q.ZeroBotApproximateVotes(ctx); err != nil {
			return fmt.Errorf("failed to zero approximate_votes on bots: %w", err)
		}
		if err := q.RecomputeBotApproximateVotes(ctx, targetType); err != nil {
			return fmt.Errorf("failed to recompute approximate_votes on bots: %w", err)
		}
	case "server":
		if err := q.ZeroServerApproximateVotes(ctx); err != nil {
			return fmt.Errorf("failed to zero approximate_votes on servers: %w", err)
		}
		if err := q.RecomputeServerApproximateVotes(ctx, targetType); err != nil {
			return fmt.Errorf("failed to recompute approximate_votes on servers: %w", err)
		}
	case "team":
		if err := q.ZeroTeamApproximateVotes(ctx); err != nil {
			return fmt.Errorf("failed to zero approximate_votes on teams: %w", err)
		}
		if err := q.RecomputeTeamApproximateVotes(ctx, targetType); err != nil {
			return fmt.Errorf("failed to recompute approximate_votes on teams: %w", err)
		}
	}

	return nil
}
