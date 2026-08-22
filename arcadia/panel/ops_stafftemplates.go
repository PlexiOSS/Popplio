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

// The staff-template catalog: pre-built answers staff pick from when
// approving/denying/etc, for both bot and server reviews (entity_type).
// Same simple catalog shape as badges — no rank/hierarchy concerns, so this
// mirrors ops_badges.go closely.
//
// ID is a plain string, not uuid.UUID, matching how the existing public GET
// /list/staff-templates route (types.StaffTemplate) already scans it.

type staffTemplateRow struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Emoji       string    `db:"emoji"`
	Tags        []string  `db:"tags"`
	Description string    `db:"description"`
	Type        string    `db:"type"`
	EntityType  string    `db:"entity_type"`
	CreatedAt   time.Time `db:"created_at"`
}

func (s *Server) updateStaffTemplates(ctx context.Context, q *types.QUpdateStaffTemplates) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check — every staff member with panel access can see
		// what templates exist, same as badges. Only writing them is gated.
		rows, err := state.Pool.Query(ctx,
			"SELECT id, name, emoji, tags, description, type, entity_type, created_at FROM staff_templates ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		templateRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[staffTemplateRow])

		if err != nil {
			return response{}, newError(err)
		}

		templates := make([]types.StaffTemplateUpsert, 0, len(templateRows))

		for _, t := range templateRows {
			templates = append(templates, types.StaffTemplateUpsert{
				ID:          t.ID,
				Name:        t.Name,
				Emoji:       t.Emoji,
				Tags:        types.NonNilStrings(t.Tags),
				Description: t.Description,
				Type:        t.Type,
				EntityType:  t.EntityType,
			})
		}

		return writeJSON(http.StatusOK, templates), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageTemplates) {
			return writeText(http.StatusForbidden, "You do not have permission to create staff templates [manage_templates]"), nil
		}

		if resp, err := validateEntityType(action.EntityType); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO staff_templates (name, emoji, tags, description, type, entity_type) VALUES ($1, $2, $3, $4, $5, $6)",
			action.Name, action.Emoji, action.Tags, action.Description, action.Type, action.EntityType)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageTemplates) {
			return writeText(http.StatusForbidden, "You do not have permission to update staff templates [manage_templates]"), nil
		}

		if resp, err := validateEntityType(action.EntityType); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM staff_templates WHERE id::text = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE staff_templates SET name = $1, emoji = $2, tags = $3, description = $4, type = $5, entity_type = $6 WHERE id::text = $7",
			action.Name, action.Emoji, action.Tags, action.Description, action.Type, action.EntityType, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageTemplates) {
			return writeText(http.StatusForbidden, "You do not have permission to delete staff templates [manage_templates]"), nil
		}

		id := q.Action.Delete.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM staff_templates WHERE id::text = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM staff_templates WHERE id::text = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No staff template action was specified")
	}
}

func validateEntityType(entityType string) (*response, error) {
	if entityType != "bot" && entityType != "server" {
		resp := writeText(http.StatusBadRequest, "entity_type must be \"bot\" or \"server\"")
		return &resp, nil
	}

	return nil, nil
}
