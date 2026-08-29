// Package get_random_servers implements GET /servers/@random — "Get Random
// Servers".
//
// Returns a list of servers from the database in random order
package get_random_servers

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Random Servers",
		Description: "Returns a list of servers from the database in random order",
		Resp: types.RandomServers{
			Servers: []types.IndexServer{},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetRandomIndexServers(d.Context)

	if err != nil {
		return resp.Err("Failed to query servers [db query]", err)
	}

	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}

	// Resolve all servers concurrently, since each server's resolution is independent
	if err := assets.ResolveIndexServers(d.Context, servers); err != nil {
		return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving index server.", err)
	}

	return uapi.HttpResponse{
		Json: types.RandomServers{
			Servers: servers,
		},
	}
}
