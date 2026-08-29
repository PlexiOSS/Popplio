// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func correspondingServerNamesDebug() string {
	out := "[\n"

	for _, name := range types.CorrespondingServerNames {
		out += fmt.Sprintf("    %q,\n", name)
	}

	return out + "]"
}

func correspondingGuild(server types.CorrespondingServer) snowflake.ID {
	switch server {
	case types.CorrespondingServerMain:
		return state.Config.Servers.Main
	case types.CorrespondingServerTesting:
		return state.Config.Servers.Testing
	case types.CorrespondingServerStaff:
		return state.Config.Servers.Staff
	default:
		return 0
	}
}

func roleExists(guildID, roleID snowflake.ID) bool {
	_, ok := dclient.Get().Caches().Role(guildID, roleID)
	return ok
}

func validateRoles(roleID string, correspondingRoles []types.Link) (*response, error) {
	roleSnow, err := snowflake.Parse(roleID)

	if err != nil {
		return nil, newError(err)
	}

	if !roleExists(state.Config.Servers.Staff, roleSnow) {
		resp := writeText(http.StatusBadRequest, "Role does not exist on the staff server")
		return &resp, nil
	}

	for _, role := range correspondingRoles {
		corrServer, err := types.CorrespondingServerFromString(role.Name)

		if err != nil {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf(
				"Server %s is not a supported corresponding role. Supported: %s",
				role.Name, correspondingServerNamesDebug(),
			))

			return &resp, nil
		}

		corrRoleSnow, err := snowflake.Parse(role.Value)

		if err != nil {
			return nil, newError(err)
		}

		guildID := correspondingGuild(corrServer)

		if !roleExists(guildID, corrRoleSnow) {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf(
				"Role %s does not exist on the server %s", corrRoleSnow, guildID,
			))

			return &resp, nil
		}
	}

	return nil, nil
}

func (s *Server) updateStaffPositions(ctx context.Context, q *types.QUpdateStaffPositions) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if q.Action.ListPositions != nil {
		positionRows, err := db.New(state.Pool).ListStaffPositionsFull(ctx)

		if err != nil {
			return response{}, newError(fmt.Errorf("Error while getting staff positions %s", err))
		}

		positions := make([]types.StaffPosition, 0, len(positionRows))

		for _, p := range positionRows {
			var links []types.Link

			if err := json.Unmarshal(p.CorrespondingRoles, &links); err != nil {
				return response{}, newError(err)
			}

			positions = append(positions, types.StaffPosition{
				ID:                 impls.UUIDString(p.ID),
				Name:               p.Name,
				RoleID:             p.RoleID,
				Perms:              types.NonNilStrings(p.Perms),
				CorrespondingRoles: types.NonNilLinks(links),
				Icon:               p.Icon,
				Index:              p.Index,
				CreatedAt:          types.NewTimestamp(p.CreatedAt.Time),
			})
		}

		return writeJSON(http.StatusOK, positions), nil
	}

	sm, err := impls.GetStaffMember(ctx, authData.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	smPerms := perms.Staff.SetFromStrings(sm.ResolvedPerms)
	smLowest := sm.Grants.Rank()

	switch {
	case q.Action.SwapIndex != nil:
		return s.swapIndex(ctx, q.Action.SwapIndex, smPerms, smLowest)
	case q.Action.SetIndex != nil:
		return s.setIndex(ctx, q.Action.SetIndex, smPerms, smLowest)
	case q.Action.CreatePosition != nil:
		return s.createPosition(ctx, q.Action.CreatePosition, smPerms, smLowest)
	case q.Action.EditPosition != nil:
		return s.editPosition(ctx, q.Action.EditPosition, smPerms, smLowest)
	case q.Action.DeletePosition != nil:
		return s.deletePosition(ctx, q.Action.DeletePosition, smPerms, smLowest)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No staff position action was specified")
	}
}

func (s *Server) swapIndex(ctx context.Context, action *types.StaffSwapIndex, smPerms perms.Set, smLowest int32) (response, error) {
	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to swap indexes of staff positions [manage_staff_roles]"), nil
	}

	var uuidA, uuidB pgtype.UUID

	if err := uuidA.Scan(action.A); err != nil {
		return response{}, newError(fmt.Errorf("Error while getting lower position %s", err))
	}

	if err := uuidB.Scan(action.B); err != nil {
		return response{}, newError(fmt.Errorf("Error while getting higher position %s", err))
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	queries := db.New(tx)

	indexA, err := queries.GetStaffPositionIndexByIDText(ctx, uuidA)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting lower position %s", err))
	}

	indexB, err := queries.GetStaffPositionIndexByIDText(ctx, uuidB)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting higher position %s", err))
	}

	if indexA == indexB {
		return writeText(http.StatusBadRequest, "Positions have the same index"), nil
	}

	if indexA <= smLowest || indexB <= smLowest {
		return writeText(http.StatusForbidden, "Either 'a' or 'b' is lower than the lowest index of the member"), nil
	}

	if err := queries.UpdateStaffPositionIndexByIDText(ctx, db.UpdateStaffPositionIndexByIDTextParams{Index: indexB, ID: uuidA}); err != nil {
		return response{}, newError(fmt.Errorf("Error while updating lower position %s", err))
	}

	if err := queries.UpdateStaffPositionIndexByIDText(ctx, db.UpdateStaffPositionIndexByIDTextParams{Index: indexA, ID: uuidB}); err != nil {
		return response{}, newError(fmt.Errorf("Error while updating higher position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) setIndex(ctx context.Context, action *types.StaffSetIndex, smPerms perms.Set, smLowest int32) (response, error) {
	id, err := uuid.Parse(action.ID)

	if err != nil {
		return response{}, newError(err)
	}

	idUUID := pgtype.UUID{Bytes: id, Valid: true}

	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to set the indexes of staff positions [manage_staff_roles]"), nil
	}

	if action.Index < 0 {
		return writeText(http.StatusBadRequest, "Index cannot be lower than 0"), nil
	}

	if action.Index <= smLowest {
		return writeText(http.StatusForbidden, "Index to set is lower than or equal to the lowest index of the staff member"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	queries := db.New(tx)

	currIndex, err := queries.GetStaffPositionIndexByID(ctx, idUUID)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting position %s", err))
	}

	if currIndex <= smLowest {
		return writeText(http.StatusForbidden, "Current index of position is lower than or equal to the lowest index of the staff member"), nil
	}

	if err := queries.ShiftStaffPositionIndexesFrom(ctx, action.Index); err != nil {
		return response{}, newError(fmt.Errorf("Error while shifting indexes %s", err))
	}

	if err := queries.UpdateStaffPositionIndexByIDText(ctx, db.UpdateStaffPositionIndexByIDTextParams{Index: action.Index, ID: idUUID}); err != nil {
		return response{}, newError(fmt.Errorf("Error while updating position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) createPosition(ctx context.Context, action *types.StaffCreatePosition, smPerms perms.Set, smLowest int32) (response, error) {
	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to create staff positions [manage_staff_roles]"), nil
	}

	if action.Index < 0 {
		return writeText(http.StatusBadRequest, "Index cannot be lower than 0"), nil
	}

	if action.Index <= smLowest {
		return writeText(http.StatusForbidden, "Index is lower than or equal to the lowest index of the staff member"), nil
	}

	if err := perms.Staff.ValidateStrings(action.Perms); err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	queries := db.New(tx)

	if err := queries.ShiftStaffPositionIndexesFrom(ctx, action.Index); err != nil {
		return response{}, newError(fmt.Errorf("Error while shifting indexes %s", err))
	}

	if resp, err := validateRoles(action.RoleID, action.CorrespondingRoles); err != nil {
		return response{}, err
	} else if resp != nil {
		return *resp, nil
	}

	correspondingRoles, err := json.Marshal(action.CorrespondingRoles)

	if err != nil {
		return response{}, newError(err)
	}

	err = queries.InsertStaffPositionFull(ctx, db.InsertStaffPositionFullParams{
		Name:               action.Name,
		Perms:              types.NonNilStrings(action.Perms),
		CorrespondingRoles: correspondingRoles,
		Icon:               action.Icon,
		RoleID:             action.RoleID,
		Index:              action.Index,
	})

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while updating position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) editPosition(ctx context.Context, action *types.StaffEditPosition, smPerms perms.Set, smLowest int32) (response, error) {
	id, err := uuid.Parse(action.ID)

	if err != nil {
		return response{}, newError(err)
	}

	idUUID := pgtype.UUID{Bytes: id, Valid: true}

	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to edit staff positions [manage_staff_roles]"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	queries := db.New(tx)

	current, err := queries.GetStaffPositionForUpdate(ctx, idUUID)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting position %s", err))
	}

	if current.Index <= smLowest {
		return writeText(http.StatusForbidden, "Index is lower than the lowest index of the member"), nil
	}

	if err := perms.Staff.ValidateStrings(action.Perms); err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	if err := perms.CheckPatch(smPerms, perms.Staff.SetFromStrings(current.Perms), perms.Staff.SetFromStrings(action.Perms)); err != nil {
		return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
	}

	if resp, err := validateRoles(action.RoleID, action.CorrespondingRoles); err != nil {
		return response{}, err
	} else if resp != nil {
		return *resp, nil
	}

	correspondingRoles, err := json.Marshal(action.CorrespondingRoles)

	if err != nil {
		return response{}, newError(err)
	}

	err = queries.UpdateStaffPositionFull(ctx, db.UpdateStaffPositionFullParams{
		Name:               action.Name,
		Perms:              types.NonNilStrings(action.Perms),
		CorrespondingRoles: correspondingRoles,
		RoleID:             action.RoleID,
		Icon:               action.Icon,
		ID:                 idUUID,
	})

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while updating position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) deletePosition(ctx context.Context, action *types.StaffDeletePosition, smPerms perms.Set, smLowest int32) (response, error) {
	id, err := uuid.Parse(action.ID)

	if err != nil {
		return response{}, newError(err)
	}

	idUUID := pgtype.UUID{Bytes: id, Valid: true}

	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to delete staff positions [manage_staff_roles]"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	queries := db.New(tx)

	current, err := queries.GetStaffPositionForUpdate(ctx, idUUID)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting position %s", err))
	}

	if current.Index <= smLowest {
		return writeText(http.StatusForbidden, "Index is lower than the lowest index of the member"), nil
	}

	if err := perms.CheckPatch(smPerms, perms.Staff.SetFromStrings(current.Perms), perms.Staff.NewSet()); err != nil {
		return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms [need to delete position]: %s", err)), nil
	}

	if err := queries.DeleteStaffPosition(ctx, idUUID); err != nil {
		return response{}, newError(fmt.Errorf("Error while deleting position %s", err))
	}

	if err := queries.ShiftStaffPositionIndexesAfter(ctx, current.Index); err != nil {
		return response{}, newError(fmt.Errorf("Error while shifting indexes %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}
