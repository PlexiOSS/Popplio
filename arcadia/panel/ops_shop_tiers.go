// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
)

func validateTier(action *types.VoteCreditTierUpsert) *response {
	if action.Cents < 0 {
		resp := writeText(http.StatusBadRequest, "Cents cannot be lower than 0")
		return &resp
	}

	if action.Votes < 0 {
		resp := writeText(http.StatusBadRequest, "Votes cannot be lower than 0")
		return &resp
	}

	if action.TargetType != "bot" && action.TargetType != "server" {
		resp := writeText(http.StatusBadRequest, "Target type must be either 'bot' or 'server'")
		return &resp
	}

	return nil
}

func dedupTierPositions(ctx context.Context, q *db.Queries, position int32, id string) error {
	return q.DedupTierPositions(ctx, db.DedupTierPositionsParams{
		Position: position,
		ID:       id,
	})
}

func (s *Server) updateVoteCreditTiers(ctx context.Context, q *types.QUpdateVoteCreditTiers) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListTiers != nil:
		tierRows, err := db.New(state.Pool).ListVoteCreditTiers(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		tiers := make([]types.VoteCreditTier, 0, len(tierRows))

		for _, t := range tierRows {
			tiers = append(tiers, types.VoteCreditTier{
				ID:         t.ID,
				TargetType: t.TargetType,
				Position:   t.Position,
				Cents:      float64(t.Cents),
				Votes:      t.Votes,
				CreatedAt:  types.NewTimestamp(t.CreatedAt.Time),
			})
		}

		return writeJSON(http.StatusOK, tiers), nil
	case q.Action.CreateTier != nil:
		action := q.Action.CreateTier

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create vote credit tiers [manage_shop]"), nil
		}

		if resp := validateTier(action); resp != nil {
			return *resp, nil
		}

		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		defer tx.Rollback(ctx)

		queries := db.New(tx)

		err = queries.InsertVoteCreditTier(ctx, db.InsertVoteCreditTierParams{
			ID:         action.ID,
			TargetType: action.TargetType,
			Position:   action.Position,
			Cents:      int32(action.Cents),
			Votes:      action.Votes,
		})

		if err != nil {
			return response{}, newError(err)
		}

		if err := dedupTierPositions(ctx, queries, action.Position, action.ID); err != nil {
			return response{}, newError(err)
		}

		if err := tx.Commit(ctx); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.EditTier != nil:
		action := q.Action.EditTier

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update vote credit tiers [manage_shop]"), nil
		}

		exists, err := db.New(state.Pool).CountVoteCreditTierByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if resp := validateTier(action); resp != nil {
			return *resp, nil
		}

		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		defer tx.Rollback(ctx)

		queries := db.New(tx)

		err = queries.UpdateVoteCreditTier(ctx, db.UpdateVoteCreditTierParams{
			Position:   action.Position,
			TargetType: action.TargetType,
			Cents:      int32(action.Cents),
			Votes:      action.Votes,
			ID:         action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		if err := dedupTierPositions(ctx, queries, action.Position, action.ID); err != nil {
			return response{}, newError(err)
		}

		if err := tx.Commit(ctx); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.DeleteTier != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete vote credit tiers [manage_shop]"), nil
		}

		id := q.Action.DeleteTier.ID

		queries := db.New(state.Pool)

		exists, err := queries.CountVoteCreditTierByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := queries.DeleteVoteCreditTier(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No vote credit tier action was specified")
	}
}
