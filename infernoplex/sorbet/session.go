// Copyright (C) 2026 NodeByte LTD

package sorbet

import (
	"context"
	"time"

	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

type session struct {
	ID         string
	Name       *string
	CreatedAt  time.Time
	Type       string
	TargetType string
	TargetID   string
	PermLimits []string
	Expiry     time.Time
}

func clearExpiredSessions(ctx context.Context) error {
	return db.New(state.Pool).DeleteExpiredSessions(ctx)
}

func sessionFromToken(ctx context.Context, token string) (*session, error) {
	if err := clearExpiredSessions(ctx); err != nil {
		return nil, err
	}

	row, err := db.New(state.Pool).GetFullSessionByToken(ctx, token)

	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	s := session{
		ID:         row.ID,
		CreatedAt:  row.CreatedAt.Time,
		Type:       row.Type,
		TargetType: row.TargetType,
		TargetID:   row.TargetID,
		PermLimits: row.PermLimits,
		Expiry:     row.Expiry.Time,
	}

	if row.Name.Valid {
		s.Name = &row.Name.String
	}

	return &s, nil
}
