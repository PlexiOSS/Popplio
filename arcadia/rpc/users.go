package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

// Banning a user's account from the platform entirely, gated by ban_users.
//
// This is a full account ban (users.banned), distinct from appBanSet's
// app_banned — a much narrower flag that only blocks submitting new
// applications. api/uapi.go already rejects every authenticated request from
// a banned user except sessions scoped "ban_exempt" (used for the ban appeal
// flow), so setting the column here is the entire enforcement story: nothing
// else needs to check it.
func banUserSet(ctx context.Context, m *types.RPCTargetReason, h Handle, banned bool) (Success, error) {
	if err := guardUser(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if _, err := db.New(state.Pool).UpdateUserBanned(ctx, db.UpdateUserBannedParams{
		UserID: m.TargetID,
		Banned: banned,
	}); err != nil {
		return Success{}, err
	}

	title := "Account Unbanned"
	description := fmt.Sprintf("<@%s> has unbanned <@%s>'s account.", h.UserID, m.TargetID)
	footer := "Welcome back!"

	if banned {
		title = "Account Banned"
		description = fmt.Sprintf("<@%s> has banned <@%s>'s account from the platform.", h.UserID, m.TargetID)
		footer = "They can still submit a ban appeal."
	}

	if err := modLogReason(title, description, footer, impls.ColourRed, m.Reason); err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
