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

func (s *Server) updateStaffTemplates(ctx context.Context, q *types.QUpdateStaffTemplates) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		templateRows, err := db.New(state.Pool).ListStaffTemplates(ctx)

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

		err := db.New(state.Pool).InsertStaffTemplate(ctx, db.InsertStaffTemplateParams{
			Name:        action.Name,
			Emoji:       action.Emoji,
			Tags:        types.NonNilStrings(action.Tags),
			Description: action.Description,
			Type:        action.Type,
			EntityType:  action.EntityType,
		})

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

		queries := db.New(state.Pool)

		exists, err := queries.CountStaffTemplateByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = queries.UpdateStaffTemplate(ctx, db.UpdateStaffTemplateParams{
			Name:        action.Name,
			Emoji:       action.Emoji,
			Tags:        types.NonNilStrings(action.Tags),
			Description: action.Description,
			Type:        action.Type,
			EntityType:  action.EntityType,
			ID:          action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageTemplates) {
			return writeText(http.StatusForbidden, "You do not have permission to delete staff templates [manage_templates]"), nil
		}

		id := q.Action.Delete.ID

		queries := db.New(state.Pool)

		exists, err := queries.CountStaffTemplateByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := queries.DeleteStaffTemplate(ctx, id); err != nil {
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
