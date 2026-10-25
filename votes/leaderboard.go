// Copyright (C) 2026 NodeByte LTD

package votes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	db "popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/dovewing"
)

const (
	LeaderboardPeriodAll   = "all"
	LeaderboardPeriodMonth = "month"

	leaderboardCacheTTL = 5 * time.Minute
)

var ErrInvalidLeaderboardPeriod = errors.New("period must be all or month")

func LeaderboardSince(ctx context.Context, period string) (pgtype.Timestamptz, error) {
	switch period {
	case "", LeaderboardPeriodAll:
		return pgtype.Timestamptz{}, nil
	case LeaderboardPeriodMonth:
		since, err := db.New(state.Pool).GetLatestAutomatedVoteReset(ctx)

		if errors.Is(err, pgx.ErrNoRows) {
			now := time.Now().UTC()
			return pgtype.Timestamptz{Time: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), Valid: true}, nil
		}

		return since, err
	default:
		return pgtype.Timestamptz{}, ErrInvalidLeaderboardPeriod
	}
}

func Leaderboard(ctx context.Context, period string, limit int) ([]types.VoterLeaderboardEntry, error) {
	if period == "" {
		period = LeaderboardPeriodAll
	}

	cacheKey := fmt.Sprintf("voter-leaderboard:%s:%d", period, limit)

	if cached, err := state.Redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var entries []types.VoterLeaderboardEntry

		if err := json.Unmarshal(cached, &entries); err == nil {
			return entries, nil
		}
	}

	since, err := LeaderboardSince(ctx, period)

	if err != nil {
		return nil, err
	}

	q := db.New(state.Pool)

	rows, err := q.GetTopVoters(ctx, db.GetTopVotersParams{Lim: int32(limit), Since: since})

	if err != nil {
		return nil, err
	}

	authors := make([]string, len(rows))

	for i, row := range rows {
		authors[i] = row.Author
	}

	favRows, err := q.GetVoterFavorites(ctx, db.GetVoterFavoritesParams{Authors: authors, Since: since})

	if err != nil {
		return nil, err
	}

	favorites := make(map[string]*types.VoterLeaderboardFavorite, len(favRows))

	for _, f := range favRows {
		favorites[f.Author] = &types.VoterLeaderboardFavorite{
			TargetType: f.TargetType,
			TargetID:   f.TargetID,
			Name:       f.ServerName,
			Avatar:     f.ServerAvatar,
			Vanity:     f.Vanity,
			Votes:      int(f.VoteCount),
		}
	}

	entries := make([]types.VoterLeaderboardEntry, len(rows))

	var wg sync.WaitGroup

	for i, row := range rows {
		entries[i] = types.VoterLeaderboardEntry{
			Rank:        int(row.Rank),
			Votes:       int(row.VoteCount),
			BotVotes:    int(row.BotVotes),
			ServerVotes: int(row.ServerVotes),
			OtherVotes:  int(row.OtherVotes),
			Favorite:    favorites[row.Author],
		}

		wg.Go(func() {
			user, err := dovewing.GetUser(ctx, row.Author, state.DovewingPlatformDiscord)

			if err != nil {
				state.Logger.Warn("Leaving voter off the leaderboard: user lookup failed", zap.String("user_id", row.Author), zap.Error(err))
				return
			}

			entries[i].User = user
		})

		if fav := entries[i].Favorite; fav != nil && fav.TargetType == "bot" {
			wg.Go(func() {
				bot, err := dovewing.GetUser(ctx, fav.TargetID, state.DovewingPlatformDiscord)

				if err != nil {
					entries[i].Favorite = nil
					return
				}

				fav.Name = bot.DisplayName

				if fav.Name == "" {
					fav.Name = bot.Username
				}

				fav.Avatar = bot.Avatar
			})
		}
	}

	wg.Wait()

	resolved := make([]types.VoterLeaderboardEntry, 0, len(entries))

	for _, e := range entries {
		if e.User != nil {
			resolved = append(resolved, e)
		}
	}

	if body, err := json.Marshal(resolved); err == nil {
		if err := state.Redis.Set(ctx, cacheKey, body, leaderboardCacheTTL).Err(); err != nil {
			state.Logger.Error("Failed to cache voter leaderboard", zap.Error(err))
		}
	}

	return resolved, nil
}

func LeaderboardRank(ctx context.Context, period, userID string) (types.VoterLeaderboardRank, error) {
	if period == "" {
		period = LeaderboardPeriodAll
	}

	cacheKey := fmt.Sprintf("voter-leaderboard-rank:%s:%s", period, userID)

	if cached, err := state.Redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var res types.VoterLeaderboardRank

		if err := json.Unmarshal(cached, &res); err == nil {
			return res, nil
		}
	}

	since, err := LeaderboardSince(ctx, period)

	if err != nil {
		return types.VoterLeaderboardRank{}, err
	}

	var res types.VoterLeaderboardRank

	if since.Valid {
		t := since.Time
		res.PeriodStart = &t
	}

	row, err := db.New(state.Pool).GetVoterRank(ctx, db.GetVoterRankParams{Author: userID, Since: since})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return types.VoterLeaderboardRank{}, err
	}

	if err == nil {
		res.Rank = int(row.Rank)
		res.Votes = int(row.VoteCount)
		res.TotalVoters = int(row.TotalVoters)
	}

	if body, err := json.Marshal(res); err == nil {
		if err := state.Redis.Set(ctx, cacheKey, body, leaderboardCacheTTL).Err(); err != nil {
			state.Logger.Error("Failed to cache voter leaderboard rank", zap.Error(err))
		}
	}

	return res, nil
}
