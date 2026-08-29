package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

// Barring a user from submitting applications, gated by ban_app_users.
//
// This is the only handler here that acts on a user rather than an entity, which
// is why it guards with userExists rather than botExists.

func appBanSet(ctx context.Context, m *types.RPCTargetReason, h Handle, banned bool) (Success, error) {
	if err := guardUser(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if err := db.New(state.Pool).UpdateUserAppBanned(ctx, db.UpdateUserAppBannedParams{
		UserID:    m.TargetID,
		AppBanned: banned,
	}); err != nil {
		return Success{}, err
	}

	title := "[Apps] Unbanned User"
	description := fmt.Sprintf("<@%s> has unbanned <@%s> from using apps.", h.UserID, m.TargetID)
	footer := "Welcome, back!"

	if banned {
		title = "[Apps] Banned User"
		description = fmt.Sprintf("<@%s> has banned <@%s> from using apps.", h.UserID, m.TargetID)
		footer = "Well done, young traveller. Sad to see you go..."
	}

	err := modLogReason(
		title,
		description,
		footer, impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
