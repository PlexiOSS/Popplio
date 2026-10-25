// Copyright (C) 2026 NodeByte LTD

package get_recent_changelog_bots

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

const recentLimit = 8

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Recently Updated Bots",
		Description: "The approved and certified bots that most recently posted a changelog entry, newest first.",
		Resp:        []types.RecentChangelogBot{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetRecentlyUpdatedChangelogBots(d.Context, recentLimit)

	if err != nil {
		return resp.Err("Error while getting recently updated bots [db fetch]", err)
	}

	bots := make([]types.RecentChangelogBot, len(rows))

	g, ctx := errgroup.WithContext(d.Context)

	for i, row := range rows {
		bots[i] = types.RecentChangelogBot{
			BotID:    row.BotID,
			LatestAt: row.LatestAt.Time,
			Updates:  row.Updates,
		}

		g.Go(func() error {
			user, err := dovewing.GetUser(ctx, row.BotID, state.DovewingPlatformDiscord)

			if err != nil {
				state.Logger.Warn("Failed to resolve recently updated bot", zap.Error(err), zap.String("bot_id", row.BotID))
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
