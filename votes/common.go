package votes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/PlexiOSS/Keel/dovewing"
)

var (
	entityVoteColsArr = dbutil.GetCols(types.EntityVote{})
	entityVoteCols    = strings.Join(entityVoteColsArr, ",")
)

type DbConn interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

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

	switch targetType {
	case "bot":
		var botType string
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT type, vote_banned FROM bots WHERE bot_id = $1", targetId).Scan(&botType, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bot not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch bot data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("bot is vote banned and cannot be voted for right now")
		}

		if botType != "approved" && botType != "certified" {
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
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT vote_banned FROM packs WHERE url = $1", targetId).Scan(&voteBanned)

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
		var name string
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT name, vote_banned FROM teams WHERE id = $1", targetId).Scan(&name, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("team not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch team data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("team is vote banned and cannot be voted for right now")
		}

		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/teams/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/teams/" + targetId,
			Name:    name,
		}, nil
	case "server":
		var name string
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT name, vote_banned FROM servers WHERE server_id = $1", targetId).Scan(&name, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch server data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("server is vote banned and cannot be voted for right now")
		}

		return &EntityInfo{
			URL:     state.Config.Sites.Frontend + "/servers/" + targetId,
			VoteURL: state.Config.Sites.Frontend + "/servers/" + targetId,
			Name:    name,
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

	switch targetType {
	case "bot":
		voteEntity.VoteCredits = true

		var premium bool
		var voteBlitzUntil pgtype.Timestamptz
		err := c.QueryRow(ctx, "SELECT premium, vote_blitz_until FROM bots WHERE bot_id = $1", targetId).Scan(&premium, &voteBlitzUntil)

		if err != nil {
			return nil, err
		}

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

		var premium bool
		var voteBlitzUntil pgtype.Timestamptz
		err := c.QueryRow(ctx, "SELECT premium, vote_blitz_until FROM servers WHERE server_id = $1", targetId).Scan(&premium, &voteBlitzUntil)

		if err != nil {
			return nil, err
		}

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
		if GetDoubleVote() {
			voteEntity.PerUser = 2
			voteEntity.VoteTime = 6
			voteEntity.WeekendBonus = true
		}
	case "pack":
		// Packs cannot be premium yet
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

	var rows pgx.Rows

	rows, err = c.Query(
		ctx,
		"SELECT "+entityVoteCols+" FROM entity_votes WHERE author = $1 AND target_id = $2 AND target_type = $3 AND void = false ORDER BY created_at DESC",
		userId,
		targetId,
		targetType,
	)

	if err != nil {
		return nil, err
	}

	validVotes, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.EntityVote])

	if errors.Is(err, pgx.ErrNoRows) {
		validVotes = []types.EntityVote{}
	} else if err != nil {
		return nil, err
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
	var upvotes int
	var downvotes int

	err := c.QueryRow(
		ctx,
		"SELECT COUNT(*) FILTER (WHERE upvote), COUNT(*) FILTER (WHERE NOT upvote) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND void = false",
		targetId, targetType,
	).Scan(&upvotes, &downvotes)

	if err != nil {
		return 0, err
	}

	return upvotes - downvotes, nil
}

func EntityGiveVotes(ctx context.Context, c DbConn, upvote bool, author, targetType, targetId string, vi *types.VoteInfo) error {
	for i := 0; i < vi.PerUser; i++ {
		_, err := c.Exec(ctx, "INSERT INTO entity_votes (author, target_id, target_type, upvote, vote_num) VALUES ($1, $2, $3, $4, $5)", author, targetId, targetType, upvote, i)

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

	switch targetType {
	case "bot":
		_, err = c.Exec(ctx, "UPDATE bots SET approximate_votes = $1 WHERE bot_id = $2", nvc, targetId)
	case "server":
		_, err = c.Exec(ctx, "UPDATE servers SET approximate_votes = $1 WHERE server_id = $2", nvc, targetId)
	case "team":
		_, err = c.Exec(ctx, "UPDATE teams SET approximate_votes = $1 WHERE id = $2", nvc, targetId)
	}

	if err != nil {
		return fmt.Errorf("failed to update vote count: %w", err)
	}

	return nil
}

// approximateVotesTargets maps a target type to the table/id-column its
// cached approximate_votes column lives on. Packs aren't included --
// EntityPostVote doesn't track them either, so there's nothing to keep in
// sync for that type.
var approximateVotesTargets = map[string]struct{ table, idCol string }{
	"bot":    {"bots", "bot_id"},
	"server": {"servers", "server_id"},
	"team":   {"teams", "id"},
}

// RecomputeApproximateVotes recalculates every entity of targetType's
// cached approximate_votes column from what's currently non-void in
// entity_votes. Zeroes every row of that type first, then adds back
// whatever's left (e.g. immutable votes that survive a reset) via the same
// upvote-minus-downvote count EntityGetVoteCount uses per-entity -- done in
// bulk here since this is meant for "every entity of a type at once"
// operations (a full vote reset), not a single entity (use EntityPostVote
// for that).
func RecomputeApproximateVotes(ctx context.Context, c DbConn, targetType string) error {
	target, ok := approximateVotesTargets[targetType]

	if !ok {
		return nil
	}

	if _, err := c.Exec(ctx, "UPDATE "+target.table+" SET approximate_votes = 0"); err != nil {
		return fmt.Errorf("failed to zero approximate_votes on %s: %w", target.table, err)
	}

	_, err := c.Exec(ctx,
		"UPDATE "+target.table+" t SET approximate_votes = v.count FROM "+
			"(SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count "+
			"FROM entity_votes WHERE target_type = $1 AND void = false GROUP BY target_id) v "+
			"WHERE t."+target.idCol+" = v.target_id",
		targetType,
	)

	if err != nil {
		return fmt.Errorf("failed to recompute approximate_votes on %s: %w", target.table, err)
	}

	return nil
}
