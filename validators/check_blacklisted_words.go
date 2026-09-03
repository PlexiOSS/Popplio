// Copyright (C) 2026 NodeByte LTD

package validators

import (
	"context"
	"errors"
	"fmt"

	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

type BlacklistedWordRow struct {
	Word    string   `db:"word"`
	Systems []string `db:"systems"`
}

func GetWordBlacklistSystems(ctx context.Context, word string) ([]string, error) {
	systems, err := db.New(state.Pool).GetWordBlacklistSystems(ctx, word)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get blacklisted word: %w", err)
	}

	return systems, nil
}
