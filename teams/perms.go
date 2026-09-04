// Copyright (C) 2026 NodeByte LTD

package teams

import (
	"context"
	"errors"
	"fmt"

	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func IsValidPerm(perm string) bool {
	return perms.Entity.Validate(perms.Perm(perm)) == nil
}

func GetEntityPerms(ctx context.Context, userId, targetType, targetId string) (perms.Set, error) {
	q := db.New(state.Pool)

	var teamUUID pgtype.UUID

	switch targetType {
	case "user":
		if targetId != userId {
			return perms.Set{}, fmt.Errorf("users do not have permissions on other users")
		}

		return perms.Entity.NewSet(perms.EntityOwner), nil
	case "bot":
		row, err := q.GetBotTeamAndOwner(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("bot not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding bot: %v", err)
		}

		if row.Owner.Valid {
			if row.Owner.String == userId {
				return perms.Entity.NewSet(perms.EntityOwner), nil
			}

			return perms.Set{}, nil
		}

		teamUUID = row.TeamOwner
	case "pack":
		owner, err := q.GetPackOwner(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("pack not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding pack: %v", err)
		}

		if owner == userId {
			return perms.Entity.NewSet(perms.EntityOwner), nil
		}

		return perms.Set{}, nil
	case "pack_emoji":
		// Pack emojis have no owner of their own -- permission on one means
		// permission on the pack it belongs to.
		row, err := q.GetPackEmojiByID(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("pack emoji not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding pack emoji: %v", err)
		}

		owner, err := q.GetPackOwner(ctx, row.PackUrl)

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding owning pack: %v", err)
		}

		if owner == userId {
			return perms.Entity.NewSet(perms.EntityOwner), nil
		}

		return perms.Set{}, nil
	case "pack_sticker":
		row, err := q.GetPackStickerByID(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("pack sticker not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding pack sticker: %v", err)
		}

		owner, err := q.GetPackOwner(ctx, row.PackUrl)

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding owning pack: %v", err)
		}

		if owner == userId {
			return perms.Entity.NewSet(perms.EntityOwner), nil
		}

		return perms.Set{}, nil
	case "pack_sound":
		row, err := q.GetPackSoundByID(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("pack sound not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding pack sound: %v", err)
		}

		owner, err := q.GetPackOwner(ctx, row.PackUrl)

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding owning pack: %v", err)
		}

		if owner == userId {
			return perms.Entity.NewSet(perms.EntityOwner), nil
		}

		return perms.Set{}, nil
	case "team":
		if _, err := uuid.Parse(targetId); err != nil {
			return perms.Set{}, fmt.Errorf("invalid team id")
		}

		if err := teamUUID.Scan(targetId); err != nil {
			return perms.Set{}, fmt.Errorf("invalid team id")
		}
	case "server":
		teamOwner, err := q.GetServerTeamOwner(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return perms.Set{}, fmt.Errorf("server not found")
		}

		if err != nil {
			return perms.Set{}, fmt.Errorf("error finding server: %v", err)
		}

		teamUUID = teamOwner
	default:
		return perms.Set{}, fmt.Errorf("invalid target type")
	}

	if !teamUUID.Valid {
		return perms.Set{}, fmt.Errorf("invalid team id")
	}

	teamPerms, err := q.GetTeamMemberFlags(ctx, db.GetTeamMemberFlagsParams{
		TeamID: teamUUID,
		UserID: userId,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return perms.Set{}, nil
	}

	if err != nil {
		return perms.Set{}, fmt.Errorf("error finding team member: %v", err)
	}

	return perms.Entity.ResolveStrings(teamPerms), nil
}
