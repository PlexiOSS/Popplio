package rpc

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	reviewassets "popplio/routes/reviews/assets"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var reviewIDPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

func deleteReview(ctx context.Context, m *types.RPCDeleteReview, h Handle) (Success, error) {
	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	raw := reviewIDPattern.FindString(m.ReviewID)

	if raw == "" {
		return Success{}, errors.New("enter a review ID or a link to the review")
	}

	var reviewID pgtype.UUID

	if err := reviewID.Scan(raw); err != nil {
		return Success{}, fmt.Errorf("invalid review ID: %w", err)
	}

	targetType := h.TargetType.String()
	q := db.New(state.Pool)

	review, err := q.GetReviewForModify(ctx, db.GetReviewForModifyParams{
		ID:         reviewID,
		TargetID:   m.TargetID,
		TargetType: targetType,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return Success{}, fmt.Errorf("no review %s on %s %s", raw, targetType, m.TargetID)
	}

	if err != nil {
		return Success{}, err
	}

	if err := q.DeleteReviewByID(ctx, reviewID); err != nil {
		return Success{}, err
	}

	if err := reviewassets.GCTrigger(m.TargetID, targetType); err != nil {
		return Success{}, fmt.Errorf("review deleted, but cleaning up its replies failed: %w", err)
	}

	description := fmt.Sprintf("<@%s> deleted a review by <@%s> (%d stars) on %s `%s`", h.UserID, review.Author, review.Stars, targetType, m.TargetID)

	if err := modLogReason("Review Deleted", description, "Replies to the review were removed too.", impls.ColourRed, m.Reason); err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
