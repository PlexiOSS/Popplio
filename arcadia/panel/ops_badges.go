package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// The badge catalog: what badges exist, their icon/color, and which entity
// types they can go on. Assigning one to an entity is a separate RPC action
// (see rpc.assignBadgeSet), not a panel op — this only manages the catalog.

type badgeRow struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Icon        string    `db:"icon"`
	Color       string    `db:"color"`
	TargetTypes []string  `db:"target_types"`
	CreatedAt   time.Time `db:"created_at"`
	CreatedBy   string    `db:"created_by"`
	LastUpdated time.Time `db:"last_updated"`
	UpdatedBy   string    `db:"updated_by"`
}

func (s *Server) updateBadges(ctx context.Context, q *types.QUpdateBadges) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx,
			"SELECT id, name, description, icon, color, target_types, created_at, created_by, last_updated, updated_by FROM badges ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		badgeRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[badgeRow])

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
				CreatedAt:   types.NewTimestamp(b.CreatedAt),
				CreatedBy:   b.CreatedBy,
				LastUpdated: types.NewTimestamp(b.LastUpdated),
				UpdatedBy:   b.UpdatedBy,
			})
		}

		return writeJSON(http.StatusOK, badges), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to create badges [manage_badges]"), nil
		}

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO badges (id, name, description, icon, color, target_types, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)",
			action.ID, action.Name, action.Description, action.Icon, action.Color, action.TargetTypes, authData.UserID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to update badges [manage_badges]"), nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM badges WHERE id = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE badges SET name = $1, description = $2, icon = $3, color = $4, target_types = $5, last_updated = NOW(), updated_by = $6 WHERE id = $7",
			action.Name, action.Description, action.Icon, action.Color, action.TargetTypes, authData.UserID, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to delete badges [manage_badges]"), nil
		}

		id := q.Action.Delete.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM badges WHERE id = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		// entity_badges references badges(id) ON DELETE CASCADE, so removing
		// a badge from the catalog also removes it from whoever had it —
		// there's no dangling "badge that no longer exists" state to guard.
		if _, err := state.Pool.Exec(ctx, "DELETE FROM badges WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No badge action was specified")
	}
}
