// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"fmt"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
)

func (s *Server) updateStaffMembers(ctx context.Context, q *types.QUpdateStaffMembers) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListMembers != nil:
		ids, err := db.New(state.Pool).ListStaffMemberIDs(ctx)

		if err != nil {
			return response{}, newError(fmt.Errorf("Error while getting staff members %s", err))
		}

		members := make([]types.StaffMember, 0, len(ids))

		for _, id := range ids {
			member, err := impls.GetStaffMember(ctx, id)

			if err != nil {
				return response{}, newError(err)
			}

			members = append(members, member)
		}

		return writeJSON(http.StatusOK, members), nil
	case q.Action.EditMember != nil:
		return s.editMember(ctx, authData.UserID, q.Action.EditMember)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No staff member action was specified")
	}
}

func (s *Server) editMember(ctx context.Context, callerID string, action *types.StaffEditMember) (response, error) {
	sm, err := impls.GetStaffMember(ctx, callerID)

	if err != nil {
		return response{}, newError(err)
	}

	target, err := impls.GetStaffMember(ctx, action.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	smPerms := perms.Staff.SetFromStrings(sm.ResolvedPerms)

	if !smPerms.Has(perms.StaffManageStaffMembers) {
		return writeText(http.StatusForbidden, "You do not have permission to edit staff members [manage_staff_members]"), nil
	}

	if target.Grants.Rank() < sm.Grants.Rank() {
		return writeText(http.StatusForbidden, "Target has a lower index than the member"), nil
	}

	if target.User.Bot {
		return writeText(http.StatusForbidden, perms.ErrBotAccount.Error()), nil
	}

	if err := perms.Staff.ValidateStrings(action.PermOverrides); err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	newResolved := perms.StaffGrants{
		Roles:  target.Grants.Roles,
		Extras: perms.ParseStrings(action.PermOverrides),
	}.Resolve()

	if err := perms.CheckPatch(smPerms, perms.Staff.SetFromStrings(target.ResolvedPerms), newResolved); err != nil {
		return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	queries := db.New(tx)

	if _, err := queries.GetStaffMemberForEdit(ctx, action.UserID); err != nil {
		return response{}, newError(fmt.Errorf("Error while getting member %s", err))
	}

	err = queries.UpdateStaffMemberEdit(ctx, db.UpdateStaffMemberEditParams{
		PermOverrides: action.PermOverrides,
		NoAutosync:    action.NoAutosync,
		Unaccounted:   action.Unaccounted,
		UserID:        action.UserID,
	})

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while updating member %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}
