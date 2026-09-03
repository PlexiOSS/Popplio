// Copyright (C) 2026 NodeByte LTD

package tasks

import (
	"context"
	"errors"
	"net/http"

	"popplio/db"
	"popplio/infernoplex/dclient"
	"popplio/perms"
	"popplio/state"
	"popplio/teams"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func TeamCleanup(ctx context.Context) error {
	rows, err := db.New(state.Pool).GetInfernoplexTeamServerMembers(ctx)

	if err != nil {
		return err
	}

	type memberKey struct {
		teamID uuid.UUID
		userID string
	}

	// A team can own more than one server, so GetInfernoplexTeamServerMembers
	// returns one row per (team, user, server). Group by (team, user) first
	// and decide once across every server that team owns -- deciding
	// per-row and removing on whichever row happens to be processed last
	// would wipe a user's entire team membership (including legitimate
	// access via other servers the team owns) just because they aren't
	// Administrator on ONE of them.
	byMember := make(map[memberKey][]string)
	order := make([]memberKey, 0)

	for _, row := range rows {
		key := memberKey{teamID: uuid.UUID(row.TeamID.Bytes), userID: row.UserID}

		if _, ok := byMember[key]; !ok {
			order = append(order, key)
		}

		byMember[key] = append(byMember[key], row.ServerID)
	}

	for _, key := range order {
		serverIDs := byMember[key]

		isAdmin := false
		inconclusive := false

		for _, serverID := range serverIDs {
			guildID, err := snowflake.Parse(serverID)

			if err != nil {
				continue
			}

			userID, err := snowflake.Parse(key.userID)

			if err != nil {
				continue
			}

			member, err := dclient.Get().Rest().GetMember(guildID, userID)

			if err != nil {
				if isNotFoundError(err) {
					// Confirmed: this user is genuinely not a member of
					// this server anymore. Doesn't rule out admin status
					// via another server the team owns, so keep checking
					// the rest before deciding.
					continue
				}

				// Rate limit, 5xx, network error, the bot losing access to
				// the guild, etc. -- we can't actually tell whether they
				// left, so don't treat silence as an answer. Leave this
				// member alone this cycle; the next run tries again.
				state.Logger.Warn("Team cleanup: could not determine membership, skipping",
					zap.Error(err), zap.String("team_id", key.teamID.String()), zap.String("user_id", key.userID), zap.String("server_id", serverID))
				inconclusive = true
				continue
			}

			guild, err := dclient.Get().Rest().GetGuild(guildID, false)

			if err != nil {
				state.Logger.Warn("Team cleanup: could not fetch guild, skipping",
					zap.Error(err), zap.String("team_id", key.teamID.String()), zap.String("user_id", key.userID), zap.String("server_id", serverID))
				inconclusive = true
				continue
			}

			if member.User.ID == guild.OwnerID || hasAdministratorRole(member.RoleIDs, guild.Roles) {
				isAdmin = true
				break
			}
		}

		if isAdmin || inconclusive {
			continue
		}

		if err := removeTeamMember(ctx, key.teamID, key.userID); err != nil {
			return err
		}
	}

	return nil
}

// isNotFoundError reports whether err is a Discord "not found" REST response
// (Unknown Member/Unknown Guild, HTTP 404) -- the only signal that actually
// means "this user isn't here anymore." Everything else (rate limits, 5xx,
// network failures, the bot losing access) is transient or ambiguous and
// must not be treated as evidence the user left.
func isNotFoundError(err error) bool {
	var restErr rest.Error

	if !errors.As(err, &restErr) {
		return false
	}

	return restErr.Response != nil && restErr.Response.StatusCode == http.StatusNotFound
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
	q := db.New(state.Pool)

	// Mirror delete_team_member's "must keep at least one global owner"
	// guard -- without it, an owner who's lost Administrator on every
	// server their team owns gets silently stripped from the team even if
	// they're the team's only owner, permanently locking out every
	// EntityOwner-gated action on it.
	userPerms, err := teams.GetEntityPerms(ctx, userID, "team", teamID.String())

	if err != nil {
		return err
	}

	if userPerms.Has(perms.EntityOwner) {
		if err := q.LockTeamOwnership(ctx, teamID.String()); err != nil {
			return err
		}

		ownerCount, err := q.CountTeamOwnersWithFlag(ctx, db.CountTeamOwnersWithFlagParams{
			TeamID: pgtype.UUID{Bytes: teamID, Valid: true},
			Flags:  []string{perms.EntityOwner.String()},
		})

		if err != nil {
			return err
		}

		if ownerCount < 2 {
			state.Logger.Warn("Team cleanup: skipping removal, would leave team with no owner",
				zap.String("team_id", teamID.String()), zap.String("user_id", userID))
			return nil
		}
	}

	return q.DeleteInfernoplexTeamMember(ctx, db.DeleteInfernoplexTeamMemberParams{
		TeamID: pgtype.UUID{Bytes: teamID, Valid: true},
		UserID: userID,
	})
}
