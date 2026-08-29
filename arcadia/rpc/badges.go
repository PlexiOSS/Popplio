package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

// Awarding and removing a badge on an entity, gated by assign_badges.
//
// This acts on entity_badges, not the badges catalog itself (that's a
// separate, catalog-only action set — see actions_badges.go /
// ops_badges.go). h.TargetType is the badge recipient's type; m.BadgeID
// names which catalog row to (un)assign.

func assignBadgeSet(ctx context.Context, m *types.RPCAssignBadge, h Handle, assign bool) (Success, error) {
	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	q := db.New(state.Pool)

	badgeExists, err := q.CountBadgeByID(ctx, m.BadgeID)

	if err != nil {
		return Success{}, err
	}

	if !badgeExists {
		return Success{}, fmt.Errorf("badge %q does not exist", m.BadgeID)
	}

	if assign {
		err := q.UpsertEntityBadge(ctx, db.UpsertEntityBadgeParams{
			TargetType: h.TargetType.String(),
			TargetID:   m.TargetID,
			BadgeID:    m.BadgeID,
			Reason:     m.Reason,
			AwardedBy:  h.UserID,
		})

		if err != nil {
			return Success{}, err
		}
	} else {
		err := q.DeleteEntityBadge(ctx, db.DeleteEntityBadgeParams{
			TargetType: h.TargetType.String(),
			TargetID:   m.TargetID,
			BadgeID:    m.BadgeID,
		})

		if err != nil {
			return Success{}, err
		}
	}

	title := "Badge Removed"
	description := fmt.Sprintf("<@%s> removed badge `%s` from %s `%s`", h.UserID, m.BadgeID, h.TargetType.String(), m.TargetID)

	if assign {
		title = "Badge Assigned"
		description = fmt.Sprintf("<@%s> assigned badge `%s` to %s `%s`", h.UserID, m.BadgeID, h.TargetType.String(), m.TargetID)
	}

	if err := modLogReason(title, description, "Badges are purely cosmetic.", impls.ColourBlurple, m.Reason); err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
