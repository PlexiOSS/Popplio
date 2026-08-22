package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

func premiumAdd(ctx context.Context, m *types.RPCPremiumAdd, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return premiumAddServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	_, err := state.Pool.Exec(ctx,
		"UPDATE bots SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE bot_id = $2",
		m.TimePeriodHours, m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Premium Added!",
		fmt.Sprintf("<@%s> has added premium to <@%s> for %d hours", h.UserID, m.TargetID, m.TimePeriodHours),
		"Well done, young traveller! Use it wisely...", impls.ColourGreen, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func premiumAddServer(ctx context.Context, m *types.RPCPremiumAdd, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	_, err := state.Pool.Exec(ctx,
		"UPDATE servers SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE server_id = $2",
		m.TimePeriodHours, m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Server Premium Added!",
		fmt.Sprintf("<@%s> has added premium to server `%s` for %d hours", h.UserID, m.TargetID, m.TimePeriodHours),
		"Well done, young traveller! Use it wisely...", impls.ColourGreen, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func premiumRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return premiumRemoveServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET premium = false WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Premium Removed!",
		fmt.Sprintf("<@%s> has removed premium from <@%s>", h.UserID, m.TargetID),
		"Well done, young traveller. Sad to see you go...", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func premiumRemoveServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET premium = false WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Server Premium Removed!",
		fmt.Sprintf("<@%s> has removed premium from server `%s`", h.UserID, m.TargetID),
		"Well done, young traveller. Sad to see you go...", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
