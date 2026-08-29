package tasks

import (
	"context"

	"popplio/db"
	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TeamCleanup(ctx context.Context) error {
	rows, err := db.New(state.Pool).GetInfernoplexTeamServerMembers(ctx)

	if err != nil {
		return err
	}

	type memberRow struct {
		teamID   uuid.UUID
		userID   string
		serverID string
	}

	memberRows := make([]memberRow, len(rows))
	for i, row := range rows {
		memberRows[i] = memberRow{
			teamID:   uuid.UUID(row.TeamID.Bytes),
			userID:   row.UserID,
			serverID: row.ServerID,
		}
	}

	for _, r := range memberRows {
		guildID, err := snowflake.Parse(r.serverID)

		if err != nil {
			continue
		}

		userID, err := snowflake.Parse(r.userID)

		if err != nil {
			continue
		}

		member, err := dclient.Get().Rest().GetMember(guildID, userID)

		if err != nil {

			if err := removeTeamMember(ctx, r.teamID, r.userID); err != nil {
				return err
			}

			continue
		}

		guild, err := dclient.Get().Rest().GetGuild(guildID, false)

		if err != nil {
			continue
		}

		isAdmin := member.User.ID == guild.OwnerID || hasAdministratorRole(member.RoleIDs, guild.Roles)

		if !isAdmin {
			if err := removeTeamMember(ctx, r.teamID, r.userID); err != nil {
				return err
			}
		}
	}

	return nil
}

func hasAdministratorRole(roleIDs []snowflake.ID, roles []discord.Role) bool {
	byID := make(map[snowflake.ID]discord.Role, len(roles))

	for _, role := range roles {
		byID[role.ID] = role
	}

	for _, id := range roleIDs {
		if role, ok := byID[id]; ok && role.Permissions.Has(discord.PermissionAdministrator) {
			return true
		}
	}

	return false
}

func removeTeamMember(ctx context.Context, teamID uuid.UUID, userID string) error {
	return db.New(state.Pool).DeleteInfernoplexTeamMember(ctx, db.DeleteInfernoplexTeamMemberParams{
		TeamID: pgtype.UUID{Bytes: teamID, Valid: true},
		UserID: userID,
	})
}
