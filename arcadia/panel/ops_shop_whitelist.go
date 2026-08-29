// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
)

func (s *Server) updateBotWhitelist(ctx context.Context, q *types.QUpdateBotWhitelist) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		whitelistRows, err := db.New(state.Pool).ListBotWhitelist(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		entries := make([]types.BotWhitelist, 0, len(whitelistRows))

		for _, w := range whitelistRows {
			entries = append(entries, types.BotWhitelist{
				BotID:     w.BotID,
				UserID:    w.UserID,
				Reason:    w.Reason,
				CreatedAt: types.NewTimestamp(w.CreatedAt.Time),
			})
		}

		return writeJSON(http.StatusOK, entries), nil
	case q.Action.Add != nil:
		action := q.Action.Add

		if !userPerms.Has(perms.StaffManageBotWhitelist) {
			return writeText(http.StatusForbidden, "You do not have permission to add to the bot whitelist [bot_whitelist.create]"), nil
		}

		err := db.New(state.Pool).InsertBotWhitelist(ctx, db.InsertBotWhitelistParams{
			UserID: authData.UserID,
			BotID:  action.BotID,
			Reason: action.Reason,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageBotWhitelist) {
			return writeText(http.StatusForbidden, "You do not have permission to update bot whitelist [bot_whitelist.update]"), nil
		}

		queries := db.New(state.Pool)

		exists, err := queries.CountBotWhitelistByBotID(ctx, action.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := queries.UpdateBotWhitelistReason(ctx, db.UpdateBotWhitelistReasonParams{
			Reason: action.Reason,
			BotID:  action.BotID,
		}); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageBotWhitelist) {
			return writeText(http.StatusForbidden, "You do not have permission to delete bot whitelist entries [bot_whitelist.delete]"), nil
		}

		botID := q.Action.Delete.BotID

		queries := db.New(state.Pool)

		exists, err := queries.CountBotWhitelistByBotID(ctx, botID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := queries.DeleteBotWhitelist(ctx, botID); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No bot whitelist action was specified")
	}
}
