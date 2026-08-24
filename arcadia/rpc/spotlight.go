package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

func spotlightAdd(ctx context.Context, m *types.RPCSpotlightAdd, h Handle) (Success, error) {
	table, idCol, err := entityTable(h.TargetType)

	if err != nil {
		return Success{}, err
	}

	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	_, err = state.Pool.Exec(ctx,
		"UPDATE "+table+" SET spotlighted_until = GREATEST(COALESCE(spotlighted_until, NOW()), NOW()) + make_interval(hours => $1) WHERE "+idCol+" = $2",
		m.TimePeriodHours, m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Spotlighted!",
		fmt.Sprintf("<@%s> has spotlighted `%s` for %d hours", h.UserID, m.TargetID, m.TimePeriodHours),
		"Shine on!", impls.ColourGreen, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func spotlightRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	table, idCol, err := entityTable(h.TargetType)

	if err != nil {
		return Success{}, err
	}

	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE "+table+" SET spotlighted_until = NULL WHERE "+idCol+" = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Unspotlighted!",
		fmt.Sprintf("<@%s> has removed the spotlight from `%s`", h.UserID, m.TargetID),
		"Back to the crowd...", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
