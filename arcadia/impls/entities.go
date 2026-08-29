package impls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// EntityManagers is the set of users who own/manage an entity.
type EntityManagers struct {
	users []manager
}

type manager struct {
	mentionable bool
	user        string
}

// All returns every manager's user id.
func (e EntityManagers) All() []string {
	all := make([]string, 0, len(e.users))
	for _, m := range e.users {
		all = append(all, m.user)
	}

	return all
}

// Mentionables returns only the managers flagged mentionable.
func (e EntityManagers) Mentionables() []string {
	out := make([]string, 0, len(e.users))
	for _, m := range e.users {
		if m.mentionable {
			out = append(out, m.user)
		}
	}

	return out
}

// MentionUsers renders the mentionable managers as a ", "-joined mention list.
func (e EntityManagers) MentionUsers() string {
	out := make([]string, 0, len(e.users))
	for _, m := range e.users {
		if m.mentionable {
			out = append(out, "<@"+m.user+">")
		}
	}

	return strings.Join(out, ", ")
}

// GetEntityManagers resolves who manages a given entity. Error strings are
// user-visible in Discord and the panel, so they are reproduced verbatim.
func GetEntityManagers(ctx context.Context, targetType types.TargetType, targetID string) (EntityManagers, error) {
	q := db.New(state.Pool)

	var teamID uuid.UUID

	switch targetType {
	case types.TargetTypeBot:
		row, err := q.GetBotTeamAndOwner(ctx, targetID)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while checking for owner of bot %s: %s", targetID, err)
		}

		if row.Owner.Valid {
			return EntityManagers{users: []manager{{mentionable: true, user: row.Owner.String}}}, nil
		}

		if !row.TeamOwner.Valid {
			return EntityManagers{}, fmt.Errorf("Bot %s is not owned by a team or a user. Please contact a dev right now!", targetID)
		}

		teamID = uuid.UUID(row.TeamOwner.Bytes)
	case types.TargetTypeServer:
		teamOwner, err := q.GetServerTeamOwner(ctx, targetID)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while checking for team owner of server %s: %s", targetID, err)
		}

		teamID = uuid.UUID(teamOwner.Bytes)
	case types.TargetTypeTeam:
		parsed, err := uuid.Parse(targetID)

		if err != nil {
			return EntityManagers{}, fmt.Errorf("Error while parsing team id %s: %s", targetID, err)
		}

		teamID = parsed
	case types.TargetTypeUser:
		exists, err := q.UserExists(ctx, targetID)

		switch {
		case err == nil && exists:
			return EntityManagers{users: []manager{{mentionable: true, user: targetID}}}, nil
		case err == nil:
			return EntityManagers{}, fmt.Errorf("User %s not found.", targetID)
		case errors.Is(err, pgx.ErrNoRows):
			return EntityManagers{}, fmt.Errorf("User %s not found.", targetID)
		default:
			return EntityManagers{}, fmt.Errorf("Error while checking for user %s: %s", targetID, err)
		}
	case types.TargetTypePack:
		return EntityManagers{}, errors.New("Packs are not supported yet!")
	default:
		return EntityManagers{}, errors.New("Packs are not supported yet!")
	}

	rows, err := q.GetTeamMembersMentionable(ctx, pgtype.UUID{Bytes: teamID, Valid: true})

	if err != nil {
		return EntityManagers{}, fmt.Errorf("Error while getting team members of team %s: %s", teamID, err)
	}

	if len(rows) == 0 {
		return EntityManagers{}, fmt.Errorf("Entity %s is on a team with no members. Please contact a dev right now!", targetID)
	}

	users := make([]manager, 0, len(rows))
	for _, m := range rows {
		users = append(users, manager{mentionable: m.Mentionable, user: m.UserID})
	}

	return EntityManagers{users: users}, nil
}

// GetBotManagers resolves managers for many bots in a fixed number of queries.
//
// GetEntityManagers costs two round trips per bot, so the panel's bot queue and
// entity search were issuing 2N queries for an N-bot response. This resolves the
// same data with three queries regardless of N.
//
// The per-bot error messages are reproduced exactly, so a caller cannot tell the
// difference between this and calling GetEntityManagers in a loop.
func GetBotManagers(ctx context.Context, botIDs []string) (map[string]EntityManagers, error) {
	out := make(map[string]EntityManagers, len(botIDs))

	if len(botIDs) == 0 {
		return out, nil
	}

	owners, err := db.New(state.Pool).GetBotsOwnersByIDs(ctx, botIDs)

	if err != nil {
		return nil, fmt.Errorf("Error while checking for owner of bots: %s", err)
	}

	byBot := make(map[string]db.GetBotsOwnersByIDsRow, len(owners))
	teamIDs := make([]uuid.UUID, 0, len(owners))

	for _, row := range owners {
		byBot[row.BotID] = row

		if !row.Owner.Valid && row.TeamOwner.Valid {
			teamIDs = append(teamIDs, uuid.UUID(row.TeamOwner.Bytes))
		}
	}

	teams, err := teamMembers(ctx, teamIDs)

	if err != nil {
		return nil, err
	}

	for _, botID := range botIDs {
		row, ok := byBot[botID]

		if !ok {
			return nil, fmt.Errorf("Error while checking for owner of bot %s: %s", botID, pgx.ErrNoRows)
		}

		if row.Owner.Valid {
			out[botID] = EntityManagers{users: []manager{{mentionable: true, user: row.Owner.String}}}
			continue
		}

		if !row.TeamOwner.Valid {
			return nil, fmt.Errorf("Bot %s is not owned by a team or a user. Please contact a dev right now!", botID)
		}

		members := teams[uuid.UUID(row.TeamOwner.Bytes)]

		if len(members) == 0 {
			return nil, fmt.Errorf("Entity %s is on a team with no members. Please contact a dev right now!", botID)
		}

		out[botID] = EntityManagers{users: members}
	}

	return out, nil
}

// GetServerManagers is the server equivalent of GetBotManagers.
func GetServerManagers(ctx context.Context, serverIDs []string) (map[string]EntityManagers, error) {
	out := make(map[string]EntityManagers, len(serverIDs))

	if len(serverIDs) == 0 {
		return out, nil
	}

	owners, err := db.New(state.Pool).GetServersTeamOwnersByIDs(ctx, serverIDs)

	if err != nil {
		return nil, fmt.Errorf("Error while checking for team owner of servers: %s", err)
	}

	byServer := make(map[string]uuid.UUID, len(owners))
	teamIDs := make([]uuid.UUID, 0, len(owners))

	for _, row := range owners {
		teamOwner := uuid.UUID(row.TeamOwner.Bytes)
		byServer[row.ServerID] = teamOwner
		teamIDs = append(teamIDs, teamOwner)
	}

	teams, err := teamMembers(ctx, teamIDs)

	if err != nil {
		return nil, err
	}

	for _, serverID := range serverIDs {
		teamID, ok := byServer[serverID]

		if !ok {
			return nil, fmt.Errorf("Error while checking for team owner of server %s: %s", serverID, pgx.ErrNoRows)
		}

		members := teams[teamID]

		if len(members) == 0 {
			return nil, fmt.Errorf("Entity %s is on a team with no members. Please contact a dev right now!", serverID)
		}

		out[serverID] = EntityManagers{users: members}
	}

	return out, nil
}

// teamMembers loads the members of many teams in one query.
func teamMembers(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID][]manager, error) {
	out := make(map[uuid.UUID][]manager, len(teamIDs))

	if len(teamIDs) == 0 {
		return out, nil
	}

	pgTeamIDs := make([]pgtype.UUID, len(teamIDs))
	for i, id := range teamIDs {
		pgTeamIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	members, err := db.New(state.Pool).GetTeamMembersByTeamIDs(ctx, pgTeamIDs)

	if err != nil {
		return nil, fmt.Errorf("Error while getting team members of teams: %s", err)
	}

	for _, m := range members {
		teamID := uuid.UUID(m.TeamID.Bytes)
		out[teamID] = append(out[teamID], manager{mentionable: m.Mentionable, user: m.UserID})
	}

	return out, nil
}

// OwnedBy is one entity a user owns.
type OwnedBy struct {
	TargetType  types.TargetType
	TargetID    string
	EntityState string
}

// GetOwnedBy unions the bots and servers owned via the user's teams, bots
// owned directly (not team-owned), and the packs they own outright. Unknown
// entity strings are logged and skipped.
func GetOwnedBy(ctx context.Context, userID string) ([]OwnedBy, error) {
	rows, err := db.New(state.Pool).GetUserOwnedEntities(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("Error while executing query for user %s: %s", userID, err)
	}

	var ownedBy []OwnedBy

	for _, row := range rows {
		id, entityState, entity := row.ID, row.Type, row.Entity

		var targetType types.TargetType

		switch entity {
		case "bot":
			targetType = types.TargetTypeBot
		case "server":
			targetType = types.TargetTypeServer
		case "pack":
			targetType = types.TargetTypePack
		default:
			state.Logger.Error("Unknown entity type encountered", zap.String("entity", entity), zap.String("userID", userID))
			continue
		}

		ownedBy = append(ownedBy, OwnedBy{
			TargetType:  targetType,
			TargetID:    id,
			EntityState: entityState,
		})
	}

	return ownedBy, nil
}

// NotifyOwners sends alert to every user ID in owners, best-effort -- a
// failed send is logged and otherwise ignored, same contract as every other
// notifications.PushNotification call site (the staff action this is
// called from has already succeeded and committed, so a notification
// failure shouldn't read back as the action itself having failed).
func NotifyOwners(owners []string, alert ptypes.Alert) {
	for _, owner := range owners {
		if err := notifications.PushNotification(owner, alert); err != nil {
			state.Logger.Warn("Failed to notify entity owner", zap.Error(err), zap.String("owner", owner), zap.String("title", alert.Title))
		}
	}
}
