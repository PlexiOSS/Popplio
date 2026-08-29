package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

func spotlightAdd(ctx context.Context, m *types.RPCSpotlightAdd, h Handle) (Success, error) {
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
		err = q.SetBotSpotlightedUntil(ctx, db.SetBotSpotlightedUntilParams{
			Hours: int32(m.TimePeriodHours),
			BotID: m.TargetID,
		})
	case types.TargetTypeServer:
		err = q.SetServerSpotlightedUntil(ctx, db.SetServerSpotlightedUntilParams{
			Hours:    int32(m.TimePeriodHours),
			ServerID: m.TargetID,
		})
	}

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
		err = q.ClearBotSpotlightedUntil(ctx, m.TargetID)
	case types.TargetTypeServer:
		err = q.ClearServerSpotlightedUntil(ctx, m.TargetID)
	}

	if err != nil {
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
