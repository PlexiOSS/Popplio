package assets

import (
	"context"
	"fmt"

	"popplio/db"
	"popplio/state"
	"popplio/types"
)

func GCTrigger(targetId, targetType string) error {
	rows, err := db.New(state.Pool).GetReviews(state.Context, db.GetReviewsParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return fmt.Errorf("failed to query reviews: %w", err)
	}

	reviews := make([]types.Review, len(rows))
	for i, row := range rows {
		reviews[i] = types.Review{
			ID:          row.ID,
			TargetType:  row.TargetType,
			TargetID:    row.TargetID,
			AuthorID:    row.Author,
			OwnerReview: row.OwnerReview,
			Content:     row.Content,
			Stars:       row.Stars,
			CreatedAt:   row.CreatedAt.Time,
			ParentID:    row.ParentID,
		}
	}

	err = GarbageCollect(state.Context, reviews)

	if err != nil {
		return fmt.Errorf("failed to garbage collect: %w", err)
	}

	return nil
}

func GarbageCollect(ctx context.Context, reviews []types.Review) error {
	var okReviews []types.Review = []types.Review{}
	var hasDeleted bool
	for i := range reviews {

		if !reviews[i].ParentID.Valid {
			okReviews = append(okReviews, reviews[i])
			continue
		}

		var found bool = false
		for j := range reviews {
			if reviews[i].ParentID.Bytes == reviews[j].ID.Bytes {
				found = true
				break
			}
		}

		if found {
			okReviews = append(okReviews, reviews[i])
		} else {
			err := db.New(state.Pool).DeleteReviewByID(ctx, reviews[i].ID)
			if err != nil {
				return err
			}

			hasDeleted = true
		}
	}

	if hasDeleted {
		return GarbageCollect(ctx, okReviews)
	}

	return nil
}
