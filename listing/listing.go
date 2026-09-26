// Copyright (C) 2026 NodeByte LTD

package listing

import (
	"context"
	"errors"

	"popplio/api"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5"
)

func IsPublic(entityType string) bool {
	return entityType == "approved" || entityType == "certified"
}

func IsStaff(ctx context.Context, userID string) bool {
	staffPerms, err := perms.StaffPerms(ctx, userID)

	if err != nil {
		return false
	}

	return staffPerms.HasAny(perms.StaffViewPanel, perms.StaffReviewEntities)
}

func CanViewUnlisted(ctx context.Context, auth uapi.AuthData, targetType, targetID string) bool {
	if !auth.Authorized {
		return false
	}

	switch auth.TargetType {
	case api.TargetTypeBot, api.TargetTypeServer:
		return auth.TargetType == targetType && auth.ID == targetID
	case api.TargetTypeUser:
	default:
		return false
	}

	if IsStaff(ctx, auth.ID) {
		return true
	}

	entityPerms, err := teams.GetEntityPerms(ctx, auth.ID, targetType, targetID)

	return err == nil && !entityPerms.IsEmpty()
}

func ViewerIsSelfOrStaff(ctx context.Context, auth uapi.AuthData, userID string) bool {
	if !auth.Authorized || auth.TargetType != api.TargetTypeUser {
		return false
	}

	return auth.ID == userID || IsStaff(ctx, auth.ID)
}

func PublicBots(bots []types.IndexBot) []types.IndexBot {
	out := make([]types.IndexBot, 0, len(bots))

	for _, b := range bots {
		if IsPublic(b.Type) {
			out = append(out, b)
		}
	}

	return out
}

func PublicServers(servers []types.IndexServer) []types.IndexServer {
	out := make([]types.IndexServer, 0, len(servers))

	for _, s := range servers {
		if IsPublic(s.Type) {
			out = append(out, s)
		}
	}

	return out
}

func EntityIsPublic(ctx context.Context, targetType, targetID string) (bool, error) {
	q := db.New(state.Pool)

	var entityType string
	var err error

	switch targetType {
	case api.TargetTypeBot:
		entityType, err = q.GetBotType(ctx, targetID)
	case api.TargetTypeServer:
		entityType, err = q.GetServerType(ctx, targetID)
	default:
		return true, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return IsPublic(entityType), nil
}
