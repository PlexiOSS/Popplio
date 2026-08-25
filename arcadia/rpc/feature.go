package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

func featureAdd(ctx context.Context, m *types.RPCFeatureAdd, h Handle) (Success, error) {
	table, idCol, err := entityTable(h.TargetType)

	if err != nil {
		return Success{}, err
	}

	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	_, err = state.Pool.Exec(ctx,
		"UPDATE "+table+" SET featured_until = GREATEST(COALESCE(featured_until, NOW()), NOW()) + make_interval(hours => $1) WHERE "+idCol+" = $2",
		m.TimePeriodHours, m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Featured!",
		fmt.Sprintf("<@%s> has featured `%s` for %d hours", h.UserID, m.TargetID, m.TimePeriodHours),
		"Shine on!", impls.ColourGreen, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func featureRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	table, idCol, err := entityTable(h.TargetType)

	if err != nil {
		return Success{}, err
	}

	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE "+table+" SET featured_until = NULL WHERE "+idCol+" = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Unfeatured!",
		fmt.Sprintf("<@%s> has removed the featured slot from `%s`", h.UserID, m.TargetID),
		"Back to the crowd...", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func entityTable(targetType types.TargetType) (table, idCol string, err error) {
	switch targetType {
	case types.TargetTypeBot:
		return "bots", "bot_id", nil
	case types.TargetTypeServer:
		return "servers", "server_id", nil
	default:
		return "", "", fmt.Errorf("featuring is only supported for bots and servers, got %q", targetType)
	}
}
