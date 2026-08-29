// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"fmt"
	"net/http"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) updateStaffDisciplinaryType(ctx context.Context, q *types.QUpdateStaffDisciplinaryType) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListDisciplinaryTypes != nil:
		typeRows, err := db.New(state.Pool).ListStaffDisciplinaryTypes(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.StaffDisciplinaryType, 0, len(typeRows))

		for _, t := range typeRows {
			out = append(out, types.StaffDisciplinaryType{
				ID:             t.ID,
				Name:           t.Name,
				Description:    t.Description,
				SelfAssignable: t.SelfAssignable,
				PermLimits:     types.NonNilStrings(t.PermLimits),
				Additory:       t.Additory,
				NeedsApproval:  t.NeedsApproval,
				MaxExpiry:      numericToFloat64Ptr(t.MaxExpiry),
				CreatedAt:      types.NewTimestamp(t.CreatedAt.Time),
			})
		}

		return writeJSON(http.StatusOK, out), nil
	case q.Action.CreateDisciplinaryType != nil:
		action := q.Action.CreateDisciplinaryType

		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to create staff disciplinary types [manage_disciplinaries]"), nil
		}

		if err := perms.Staff.ValidateStrings(action.PermLimits); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		if err := perms.CheckPatch(userPerms, perms.Staff.NewSet(), perms.Staff.SetFromStrings(action.PermLimits)); err != nil {
			return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
		}

		err := db.New(state.Pool).InsertStaffDisciplinaryType(ctx, db.InsertStaffDisciplinaryTypeParams{
			ID:             action.ID,
			Name:           action.Name,
			Description:    action.Description,
			SelfAssignable: action.SelfAssignable,
			PermLimits:     types.NonNilStrings(action.PermLimits),
			Additory:       action.Additory,
			NeedsApproval:  action.NeedsApproval,
			Secs:           float8FromPtr(action.MaxExpiry),
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.EditDisciplinaryType != nil:
		action := q.Action.EditDisciplinaryType

		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to update staff disciplinary types [manage_disciplinaries]"), nil
		}

		if err := perms.Staff.ValidateStrings(action.PermLimits); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		if err := perms.CheckPatch(userPerms, perms.Staff.NewSet(), perms.Staff.SetFromStrings(action.PermLimits)); err != nil {
			return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
		}

		queries := db.New(state.Pool)

		exists, err := queries.CountStaffDisciplinaryTypeByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = queries.UpdateStaffDisciplinaryType(ctx, db.UpdateStaffDisciplinaryTypeParams{
			Name:           action.Name,
			Description:    action.Description,
			SelfAssignable: action.SelfAssignable,
			PermLimits:     types.NonNilStrings(action.PermLimits),
			Additory:       action.Additory,
			NeedsApproval:  action.NeedsApproval,
			Secs:           float8FromPtr(action.MaxExpiry),
			ID:             action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.DeleteDisciplinaryType != nil:
		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to delete staff disciplinary types [manage_disciplinaries]"), nil
		}

		id := q.Action.DeleteDisciplinaryType.ID

		queries := db.New(state.Pool)

		exists, err := queries.CountStaffDisciplinaryTypeByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := queries.DeleteStaffDisciplinaryType(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No disciplinary type action was specified")
	}
}

func numericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}

	f, err := n.Float64Value()

	if err != nil || !f.Valid {
		return nil
	}

	return &f.Float64
}
