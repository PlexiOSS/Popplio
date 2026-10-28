// Copyright (C) 2026 NodeByte LTD

package bumps

import (
	"context"
	"errors"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const Cooldown = 2 * time.Hour

var (
	ErrNotFound  = errors.New("server not listed")
	ErrNotPublic = errors.New("server not public")
)

type CooldownError struct {
	Next time.Time
}

func (e *CooldownError) Error() string {
	return "bumped too recently"
}

func Bump(ctx context.Context, serverID, userID string) (time.Time, error) {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return time.Time{}, err
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	bumpedAt, err := q.BumpServer(ctx, serverID)

	if errors.Is(err, pgx.ErrNoRows) {
		row, stateErr := q.GetServerBumpState(ctx, serverID)

		switch {
		case errors.Is(stateErr, pgx.ErrNoRows):
			return time.Time{}, ErrNotFound
		case stateErr != nil:
			return time.Time{}, stateErr
		case row.State != "public" || (row.Type != "approved" && row.Type != "certified"):
			return time.Time{}, ErrNotPublic
		case row.BumpedAt.Valid:
			return time.Time{}, &CooldownError{Next: row.BumpedAt.Time.Add(Cooldown)}
		}

		return time.Time{}, ErrNotPublic
	}

	if err != nil {
		return time.Time{}, err
	}

	if err := q.InsertServerBump(ctx, db.InsertServerBumpParams{ServerID: serverID, UserID: userID}); err != nil {
		return time.Time{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, err
	}

	return bumpedAt.Time.Add(Cooldown), nil
}

func Info(ctx context.Context, serverID string, topN int) (*types.ServerBumpInfo, error) {
	q := db.New(state.Pool)

	row, err := q.GetServerBumpState(ctx, serverID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	info := &types.ServerBumpInfo{TopBumpers: []types.ServerBumper{}}

	if row.BumpedAt.Valid {
		at := row.BumpedAt.Time
		info.BumpedAt = &at

		if next := at.Add(Cooldown); next.After(time.Now()) {
			info.NextBumpAt = &next
		}
	}

	count, err := q.CountServerBumpsSince(ctx, db.CountServerBumpsSinceParams{
		ServerID: serverID,
		Since:    pgtype.Timestamptz{Time: time.Now().Add(-30 * 24 * time.Hour), Valid: true},
	})

	if err != nil {
		return nil, err
	}

	info.Bumps30d = int(count)

	top, err := q.GetServerTopBumpers(ctx, db.GetServerTopBumpersParams{ServerID: serverID, Lim: int32(topN)})

	if err != nil {
		return nil, err
	}

	for _, t := range top {
		user, err := dovewing.GetUser(ctx, t.UserID, state.DovewingPlatformDiscord)

		if err != nil {
			continue
		}

		info.TopBumpers = append(info.TopBumpers, types.ServerBumper{User: user, Bumps: int(t.Bumps)})
	}

	return info, nil
}
