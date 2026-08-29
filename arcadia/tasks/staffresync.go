package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// stringsToUUIDs converts canonical UUID strings (as produced by
// impls.UUIDString/UUIDStrings) back into pgtype.UUID for query params
// against uuid[] columns.
func stringsToUUIDs(ids []string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, len(ids))

	for i, id := range ids {
		if err := out[i].Scan(id); err != nil {
			return nil, fmt.Errorf("invalid position id %q: %w", id, err)
		}
	}

	return out, nil
}

type cachedPosition struct {
	ID                 string
	Name               string
	RoleID             string
	Index              int32
	Perms              []string
	CorrespondingRoles []types.Link
}

func (c cachedPosition) String() string {
	return fmt.Sprintf("%s [%s] (<@&%s>)", c.ID, c.Name, c.RoleID)
}

func StaffResync(ctx context.Context) error {
	if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Staff); !ok {
		return fmt.Errorf("Failed to get staff guild for staff perms resync")
	}

	type guildMember struct {
		UserID snowflake.ID
		Roles  []string
		IsBot  bool
	}

	var staffResync []guildMember

	dclient.Get().Caches().MembersForEach(state.Config.Servers.Staff, func(member discord.Member) {
		roles := make([]string, 0, len(member.RoleIDs))

		for _, roleID := range member.RoleIDs {
			roles = append(roles, roleID.String())
		}

		staffResync = append(staffResync, guildMember{UserID: member.User.ID, Roles: roles, IsBot: member.User.Bot})
	})

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	txq := db.New(tx)

	positionRows, err := txq.GetAllStaffPositions(ctx)

	if err != nil {
		return fmt.Errorf("Error while getting staff positions: %v", err)
	}

	posByID := make(map[string]cachedPosition, len(positionRows))
	posByRoleID := make(map[string]cachedPosition, len(positionRows))
	posByName := make(map[string]cachedPosition, len(positionRows))

	for _, p := range positionRows {
		var links []types.Link

		if err := json.Unmarshal(p.CorrespondingRoles, &links); err != nil {
			return err
		}

		pos := cachedPosition{
			ID:                 impls.UUIDString(p.ID),
			Name:               p.Name,
			RoleID:             p.RoleID,
			Index:              p.Index,
			Perms:              p.Perms,
			CorrespondingRoles: links,
		}

		posByID[pos.ID] = pos
		posByRoleID[pos.RoleID] = pos
		posByName[pos.Name] = pos
	}

	staff, err := txq.GetAllStaffMembersForUpdate(ctx)

	if err != nil {
		return fmt.Errorf("Error while getting staff members: %v", err)
	}

	overridePerms := make(map[string][]perms.Perm, len(staff))
	noAutosync := make(map[string]struct{})
	knownUnaccounted := make(map[string]struct{})
	unaccountedUserIDs := make(map[string]struct{})
	memberPosCache := make(map[string][]string)

	for _, member := range staff {
		overridePerms[member.UserID] = perms.ParseStrings(member.PermOverrides)

		if member.NoAutosync {
			noAutosync[member.UserID] = struct{}{}
			continue
		}

		if member.Unaccounted {
			knownUnaccounted[member.UserID] = struct{}{}
		}

		unaccountedUserIDs[member.UserID] = struct{}{}
		memberPosCache[member.UserID] = impls.UUIDStrings(member.Positions)
	}

	bots := make(map[string]struct{})

	for _, user := range staffResync {
		userID := user.UserID.String()

		if user.IsBot {
			bots[userID] = struct{}{}
			continue
		}

		if _, skip := noAutosync[userID]; skip {
			continue
		}

		dbPositions, isOnDB := memberPosCache[userID]

		currentPositions := make(map[string]struct{})

		for _, posID := range dbPositions {
			if _, ok := posByID[posID]; !ok {
				var posUUID pgtype.UUID
				if err := posUUID.Scan(posID); err != nil {
					return fmt.Errorf("invalid position id %q: %w", posID, err)
				}

				err := txq.RemoveStaffMemberPosition(ctx, db.RemoveStaffMemberPositionParams{
					ArrayRemove: posUUID,
					UserID:      userID,
				})

				if err != nil {
					return fmt.Errorf("Error while removing staff member position: %v", err)
				}

				continue
			}

			currentPositions[posID] = struct{}{}
		}

		rolePositions := make(map[string]struct{})

		if isOwner(user.UserID) {
			if ownerPos, ok := posByName["owner"]; ok {
				rolePositions[ownerPos.ID] = struct{}{}
			}
		}

		for _, role := range user.Roles {
			pos, ok := posByRoleID[role]

			if !ok {
				continue
			}

			if pos.Name == "owner" {
				continue
			}

			rolePositions[pos.ID] = struct{}{}
		}

		if !differs(rolePositions, currentPositions) {
			delete(unaccountedUserIDs, userID)
			continue
		}

		newPositionIDs := sortedKeys(rolePositions)
		oldPositionIDs := sortedKeys(currentPositions)

		exists, err := txq.UserExistsCheck(ctx, userID)

		if err != nil {
			return fmt.Errorf("Error while checking if user exists: %v", err)
		}

		if !exists {
			err := txq.InsertUserWithToken(ctx, db.InsertUserWithTokenParams{
				UserID:   userID,
				ApiToken: impls.GenRandom(512),
			})

			if err != nil {
				return fmt.Errorf("Error while inserting user: %v", err)
			}
		}

		newPositionUUIDs, err := stringsToUUIDs(newPositionIDs)

		if err != nil {
			return err
		}

		if isOnDB {
			err = txq.UpdateStaffMemberPositions(ctx, db.UpdateStaffMemberPositionsParams{
				Positions: newPositionUUIDs,
				UserID:    userID,
			})

			if err != nil {
				return fmt.Errorf("Error while updating staff member positions: %v", err)
			}
		} else {
			err = txq.InsertStaffMemberPositions(ctx, db.InsertStaffMemberPositionsParams{
				UserID:    userID,
				Positions: newPositionUUIDs,
			})

			if err != nil {
				return fmt.Errorf("Error while inserting staff member positions: %v", err)
			}
		}

		oldSP := buildPermissions(posByID, oldPositionIDs, overridePerms[userID])
		newSP := buildPermissions(posByID, newPositionIDs, overridePerms[userID])

		announceResync(userID, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "Staff Permissions Resync",
				Description: fmt.Sprintf("Updated staff permissions for <@%s>", userID),
				Fields: []discord.EmbedField{
					{Name: "Old Positions", Value: renderPositions(posByID, oldPositionIDs), Inline: impls.InlineFalse()},
					{Name: "New Positions", Value: renderPositions(posByID, newPositionIDs), Inline: impls.InlineFalse()},
					{Name: "Old Permissions", Value: renderPerms(oldSP.Resolve()), Inline: impls.InlineFalse()},
					{Name: "New Permissions", Value: renderPerms(newSP.Resolve()), Inline: impls.InlineFalse()},
				},
			}},
		})

		if err := modifyCorrespondingRoles(posByID, user.UserID, oldPositionIDs, newPositionIDs); err != nil {
			return err
		}

		delete(unaccountedUserIDs, userID)
	}

	for _, userID := range sortedKeys(unaccountedUserIDs) {
		if _, skip := noAutosync[userID]; skip {
			continue
		}

		if _, skip := knownUnaccounted[userID]; skip {
			continue
		}

		remove := len(overridePerms[userID]) == 0

		if remove {
			if err := txq.DeleteStaffMember(ctx, userID); err != nil {
				return fmt.Errorf("Error while removing unaccounted staff member: %v", err)
			}
		} else {
			err := txq.MarkStaffMemberUnaccounted(ctx, userID)

			if err != nil {
				return fmt.Errorf("Error while updating unaccounted staff member: %v", err)
			}
		}

		oldPositions, ok := memberPosCache[userID]

		if !ok {
			state.Logger.Warn("Unaccounted staff member missing from position cache", zap.String("userID", userID))
		}

		oldSP := buildPermissions(posByID, oldPositions, overridePerms[userID])

		_, isBot := bots[userID]

		verb, reason := "Removed", "they are no longer in the staff server"

		if isBot {
			reason = "bot accounts cannot hold staff permissions"
		}

		if !remove {
			verb, reason = "Updated", reason+", but they have permission overrides"
		}

		description := fmt.Sprintf("%s unaccounted staff member <@%s> as %s.", verb, userID, reason)

		announceResync(userID, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "Staff Permissions Resync",
				Description: description,
				Fields: []discord.EmbedField{
					{Name: "Old Positions", Value: renderPositions(posByID, oldPositions), Inline: impls.InlineFalse()},
					{Name: "Old Permissions", Value: renderPerms(oldSP.Resolve()), Inline: impls.InlineFalse()},
				},
			}},
		})

		userSnow, err := snowflake.Parse(userID)

		if err != nil {
			state.Logger.Warn("Unaccounted staff member has an unparseable id", zap.String("userID", userID))
			continue
		}

		if err := modifyCorrespondingRoles(posByID, userSnow, oldPositions, nil); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Error while committing transaction: %v", err)
	}

	return nil
}
