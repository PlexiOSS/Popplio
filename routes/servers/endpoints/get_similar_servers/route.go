// Package get_similar_servers implements GET /servers/{id}/similar — "Get
// Similar Servers".
//
// Returns other approved/certified, publicly listed servers sharing at
// least one tag with this server, ranked by how many tags they share
package get_similar_servers

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Similar Servers",
		Description: "Returns other approved/certified, publicly listed servers sharing at least one tag with this server, ranked by how many tags they share",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: []types.IndexServer{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	serverId := chi.URLParam(r, "id")

	if serverId == "" {
		return resp.BadRequest("id is required")
	}

	rows, err := db.New(state.Pool).GetSimilarServers(d.Context, serverId)

	if err != nil {
		return resp.Err("Error while getting similar servers [db fetch]", err)
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

	if err := assets.ResolveIndexServers(d.Context, servers); err != nil {
		return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving index server.", err)
	}

	return uapi.HttpResponse{
		Json: servers,
	}
}
