package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func certifyAdd(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return certifyAddServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if err := db.New(state.Pool).CertifyBot(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Force Certified!",
		fmt.Sprintf("<@%s> has force-certified <@%s>", h.UserID, m.TargetID),
		"Neat", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	notifyCertifyOwners(ctx, types.TargetTypeBot, m.TargetID, true)

	return NoContent(), nil
}

func certifyAddServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if err := db.New(state.Pool).CertifyServer(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Server Force Certified!",
		fmt.Sprintf("<@%s> has force-certified server `%s`", h.UserID, m.TargetID),
		"Neat", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	notifyCertifyOwners(ctx, types.TargetTypeServer, m.TargetID, true)

	return NoContent(), nil
}

func certifyRemove(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return certifyRemoveServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if err := db.New(state.Pool).UncertifyBot(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Uncertified!",
		fmt.Sprintf("<@%s> has uncertified <@%s>", h.UserID, m.TargetID),
		"Uh oh, looks like you've been naughty...", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	notifyCertifyOwners(ctx, types.TargetTypeBot, m.TargetID, false)

	return NoContent(), nil
}

func certifyRemoveServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	if err := db.New(state.Pool).UncertifyServer(ctx, m.TargetID); err != nil {
		return Success{}, err
	}

	err := modLogReason(
		"Server Uncertified!",
		fmt.Sprintf("<@%s> has uncertified server `%s`", h.UserID, m.TargetID),
		"Uh oh, looks like you've been naughty...", impls.ColourRedLower, m.Reason)

	if err != nil {
		return Success{}, err
	}

	notifyCertifyOwners(ctx, types.TargetTypeServer, m.TargetID, false)

	return NoContent(), nil
}

// notifyCertifyOwners tells an entity's owners about a certification change.
// Best-effort: fetching the owner list or sending the alert failing is
// logged (inside GetEntityManagers/NotifyOwners) and otherwise ignored --
// the certification change itself has already succeeded and committed.
func notifyCertifyOwners(ctx context.Context, targetType types.TargetType, targetID string, added bool) {
	owners, err := impls.GetEntityManagers(ctx, targetType, targetID)

	if err != nil {
		state.Logger.Warn("Failed to fetch entity managers for certification notification", zap.Error(err), zap.String("targetID", targetID))
		return
	}

	entityLabel := "bot"
	urlPath := "bots"

	if targetType == types.TargetTypeServer {
		entityLabel = "server"
		urlPath = "servers"
	}

	alert := ptypes.Alert{
		URL:      pgtype.Text{String: fmt.Sprintf("%s/%s/%s", state.Config.Sites.Frontend, urlPath, targetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	}

	if added {
		alert.Type = ptypes.AlertTypeSuccess
		alert.Title = "Certified!"
		alert.Message = "Your " + entityLabel + " has been certified."
	} else {
		alert.Type = ptypes.AlertTypeWarning
		alert.Title = "Certification Removed"
		alert.Message = "Your " + entityLabel + "'s certification has been removed."
	}

	impls.NotifyOwners(owners.All(), alert)
}
