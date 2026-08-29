package get_all_servers

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 12

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Servers",
		Description: "Gets all servers on the list. Returns a set of paginated ``IndexServer`` objects",
		Resp:        types.PagedResult[[]types.IndexServer]{},
		RespName:    "PagedResultIndexServer",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "sort",
				Description: "Sort order. Omit for newest-first; \"trending\" ranks by net votes in the last 7 days instead, and only returns servers with at least one vote in that window.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage
	trending := r.URL.Query().Get("sort") == "trending"

	q := db.New(state.Pool)

	var servers []types.IndexServer

	if trending {
		rows, err := q.GetTrendingIndexServers(d.Context, db.GetTrendingIndexServersParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})

		if err != nil {
			return resp.Err("Failed to query servers [db query]", err, zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
		}

		servers = make([]types.IndexServer, len(rows))
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
	} else {
		rows, err := q.GetIndexServersPaged(d.Context, db.GetIndexServersPagedParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})

		if err != nil {
			return resp.Err("Failed to query servers [db query]", err, zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
		}

		servers = make([]types.IndexServer, len(rows))
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
	}

	if err := assets.ResolveIndexServers(d.Context, servers); err != nil {
		return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving index server.", err)
	}

	var countRaw int64

	if trending {
		countRaw, err = q.CountTrendingServers(d.Context)
	} else {
		countRaw, err = q.CountServers(d.Context)
	}

	if err != nil {
		return resp.Err("Failed to query servers [db count]", err, zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	count := uint64(countRaw)

	data := types.PagedResult[[]types.IndexServer]{
		Count:   count,
		Results: servers,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
