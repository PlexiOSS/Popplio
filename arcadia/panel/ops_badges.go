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

// The badge catalog: what badges exist, their icon/color, and which entity
// types they can go on. Assigning one to an entity is a separate RPC action
// (see rpc.assignBadgeSet), not a panel op — this only manages the catalog.

func (s *Server) updateBadges(ctx context.Context, q *types.QUpdateBadges) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		badgeRows, err := db.New(state.Pool).ListBadges(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		badges := make([]types.Badge, 0, len(badgeRows))

		for _, b := range badgeRows {
			badges = append(badges, types.Badge{
				ID:          b.ID,
				Name:        b.Name,
				Description: b.Description,
				Icon:        b.Icon,
				Color:       b.Color,
				TargetTypes: types.NonNilStrings(b.TargetTypes),
				CreatedAt:   types.NewTimestamp(b.CreatedAt.Time),
				CreatedBy:   b.CreatedBy,
				LastUpdated: types.NewTimestamp(b.LastUpdated.Time),
				UpdatedBy:   b.UpdatedBy,
			})
		}

		return writeJSON(http.StatusOK, badges), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to create badges [manage_badges]"), nil
		}

		err := db.New(state.Pool).InsertBadge(ctx, db.InsertBadgeParams{
			ID:          action.ID,
			Name:        action.Name,
			Description: action.Description,
			Icon:        action.Icon,
			Color:       action.Color,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			CreatedBy:   authData.UserID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to update badges [manage_badges]"), nil
		}

		exists, err := db.New(state.Pool).CountBadgeByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = db.New(state.Pool).UpdateBadge(ctx, db.UpdateBadgeParams{
			Name:        action.Name,
			Description: action.Description,
			Icon:        action.Icon,
			Color:       action.Color,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			UpdatedBy:   authData.UserID,
			ID:          action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to delete badges [manage_badges]"), nil
		}

		id := q.Action.Delete.ID

		exists, err := db.New(state.Pool).CountBadgeByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := db.New(state.Pool).DeleteBadge(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No badge action was specified")
	}
}
