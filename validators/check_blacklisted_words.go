package validators

import (
	"context"
	"errors"
	"fmt"

	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// BlacklistedWordRow describes blacklisted_words's columns actually relied
// on by Popplio (it's otherwise queried via sqlc, not scanned into this
// struct in practice) -- exists so cmd/migrate's schema validation has
// something to check this table's columns against.
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
