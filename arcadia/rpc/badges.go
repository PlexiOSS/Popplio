package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
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

	var badgeExists bool

	if err := state.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM badges WHERE id = $1)", m.BadgeID).Scan(&badgeExists); err != nil {
		return Success{}, err
	}

	if !badgeExists {
		return Success{}, fmt.Errorf("badge %q does not exist", m.BadgeID)
	}

	if assign {
		_, err := state.Pool.Exec(ctx,
			"INSERT INTO entity_badges (target_type, target_id, badge_id, reason, awarded_by) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (target_type, target_id, badge_id) DO UPDATE SET reason = $4, awarded_by = $5, created_at = NOW()",
			h.TargetType.String(), m.TargetID, m.BadgeID, m.Reason, h.UserID)

		if err != nil {
			return Success{}, err
		}
	} else {
		_, err := state.Pool.Exec(ctx,
			"DELETE FROM entity_badges WHERE target_type = $1 AND target_id = $2 AND badge_id = $3",
			h.TargetType.String(), m.TargetID, m.BadgeID)

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
