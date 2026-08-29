package assets

import (
	"context"
	"errors"
	"fmt"

	"popplio/db"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Nest(ctx context.Context, id string) (int, error) {
	var depth int

	var reachedRoot bool

	q := db.New(state.Pool)

	for !reachedRoot {
		var idUUID pgtype.UUID
		if err := idUUID.Scan(id); err != nil {
			return depth, fmt.Errorf("invalid review id %s: %w", id, err)
		}

		parent, err := q.GetReviewParentID(ctx, idUUID)

		if errors.Is(err, pgx.ErrNoRows) {
			return depth, nil
		}

		if err != nil {
			return depth, fmt.Errorf("failed to query parent_id of id %s: %w", id, err)
		}

		if !parent.Valid {
			reachedRoot = true
		} else {
			id = uuid.UUID(parent.Bytes).String()
			depth++
		}
	}

	return depth, nil
}
