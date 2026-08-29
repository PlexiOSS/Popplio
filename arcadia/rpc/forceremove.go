package rpc

import (
	"context"
	"errors"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
)

// Deleting a bot, server, or pack from the list outright, gated by
// force_remove_entities.
//
// This is the one action in the package that cannot be undone, hence the
// protected-bots list: the bots the platform itself runs on cannot be kicked out
// of the main server by a mistyped id.

func forceRemove(ctx context.Context, m *types.RPCForceRemove, h Handle) (Success, error) {
	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	// Kick and the protected-bots list are a bot-only concept: neither a
	// server nor a pack has a "leave the main guild" action to take.
	if h.TargetType == types.TargetTypeBot {
		targetSnow, err := snowflake.Parse(m.TargetID)

		if err != nil {
			return Success{}, err
		}

		if m.Kick && isProtectedBot(targetSnow) {
			return Success{}, errors.New("You can't force delete this bot with 'kick' enabled!")
		}

		if err := db.New(state.Pool).DeleteBotByID(ctx, m.TargetID); err != nil {
			return Success{}, err
		}

		if err := modLogReason(
			"Force Deleted!",
			fmt.Sprintf("<@%s> has force-removed <@%s> for violating our rules or Discord ToS", h.UserID, m.TargetID),
			"Remember: don't abuse our services!", impls.ColourRed, m.Reason); err != nil {
			return Success{}, err
		}

		if m.Kick && impls.MemberOnGuild(state.Config.Servers.Main, targetSnow) {
			if err := impls.KickMember(state.Config.Servers.Main, targetSnow, "Force deleted via RPC with kick set to true"); err != nil {
				return Success{}, err
			}
		}

		return NoContent(), nil
	}

	q := db.New(state.Pool)

	var err error
	switch h.TargetType {
	case types.TargetTypeServer:
		err = q.DeleteServerByID(ctx, m.TargetID)
	case types.TargetTypePack:
		err = q.DeletePack(ctx, m.TargetID)
	default:
		return Success{}, fmt.Errorf("force removal does not support target type %s", h.TargetType)
	}

	if err != nil {
		return Success{}, err
	}

	if err := modLogReason(
		"Force Deleted!",
		fmt.Sprintf("<@%s> has force-removed %s `%s` for violating our rules or ToS", h.UserID, h.TargetType.String(), m.TargetID),
		"Remember: don't abuse our services!", impls.ColourRed, m.Reason); err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func isProtectedBot(id snowflake.ID) bool {
	for _, protected := range state.Config.Arcadia.ProtectedBots {
		if protected == id {
			return true
		}
	}

	return false
}
