package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

func featureAdd(ctx context.Context, m *types.RPCFeatureAdd, h Handle) (Success, error) {
	if _, _, err := entityTable(h.TargetType); err != nil {
		return Success{}, err
	}

	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	q := db.New(state.Pool)

	var err error
	switch h.TargetType {
	case types.TargetTypeBot:
		err = q.ApplyBotFeaturedSlot(ctx, db.ApplyBotFeaturedSlotParams{
			Hours: int32(m.TimePeriodHours),
			BotID: m.TargetID,
		})
	case types.TargetTypeServer:
		err = q.ApplyServerFeaturedSlot(ctx, db.ApplyServerFeaturedSlotParams{
			Hours:    int32(m.TimePeriodHours),
			ServerID: m.TargetID,
		})
	}

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
	if _, _, err := entityTable(h.TargetType); err != nil {
		return Success{}, err
	}

	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	q := db.New(state.Pool)

	var err error
	switch h.TargetType {
	case types.TargetTypeBot:
		err = q.ClearBotFeaturedUntil(ctx, m.TargetID)
	case types.TargetTypeServer:
		err = q.ClearServerFeaturedUntil(ctx, m.TargetID)
	}

	if err != nil {
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
