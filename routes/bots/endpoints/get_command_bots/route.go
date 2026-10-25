// Copyright (C) 2026 NodeByte LTD

package get_command_bots

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

const botsLimit = 48

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bots With Documented Commands",
		Description: "Approved and certified bots that have documented commands, most-voted first, with their command count and a preview of command names.",
		Resp:        []types.BotWithCommands{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetBotsWithCommands(d.Context, botsLimit)

	if err != nil {
		return resp.Err("Error while getting bots with commands [db fetch]", err)
	}

	bots := make([]types.BotWithCommands, len(rows))

	g, ctx := errgroup.WithContext(d.Context)

	for i, row := range rows {
		preview := row.Preview
		if preview == nil {
			preview = []string{}
		}

		bots[i] = types.BotWithCommands{
			BotID:    row.BotID,
			Commands: row.Commands,
			Preview:  preview,
		}

		g.Go(func() error {
			user, err := dovewing.GetUser(ctx, row.BotID, state.DovewingPlatformDiscord)

			if err != nil {
				state.Logger.Warn("Failed to resolve bot with commands", zap.Error(err), zap.String("bot_id", row.BotID))
				return nil
			}

			bots[i].User = user
			return nil
		})
	}

	_ = g.Wait()

	return uapi.HttpResponse{
		Json: bots,
	}
}
