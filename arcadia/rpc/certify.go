package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

func certifyAdd(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return certifyAddServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'certified' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		" Force Certified!",
		fmt.Sprintf("<@%s> has force-certified <@%s>", h.UserID, m.TargetID),
		"Neat", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func certifyAddServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET type = 'certified' WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Server Force Certified!",
		fmt.Sprintf("<@%s> has force-certified server `%s`", h.UserID, m.TargetID),
		"Neat", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func certifyRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return certifyRemoveServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'approved' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		" Uncertified!",
		fmt.Sprintf("<@%s> has uncertified <@%s>", h.UserID, m.TargetID),
		"Uh oh, looks like you've been naughty...", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func certifyRemoveServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET type = 'approved' WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Server Uncertified!",
		fmt.Sprintf("<@%s> has uncertified server `%s`", h.UserID, m.TargetID),
		"Uh oh, looks like you've been naughty...", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
