// Package get_all_bots implements GET /bots/@all — "Get All Bots".
//
// Gets approved and certified bots, filtered and sorted. Returns a set of
// paginated `IndexBot` objects
package get_all_bots

import (
	"net/http"
	"strings"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const (
	perPage      = 12
	maxTagLength = 64
)

var sorts = map[string]bool{
	"new":      true,
	"trending": true,
	"votes":    true,
	"servers":  true,
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bots",
		Description: "Gets approved and certified bots, filtered and sorted. Returns a set of paginated ``IndexBot`` objects",
		Resp:        types.PagedResult[[]types.IndexBot]{},
		RespName:    "PagedResultIndexBot",
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
				Description: "Sort order: new (default; newest first, with boosted listings on top), trending (net votes in the last 7 days, only bots with at least one vote in that window), votes (most votes this cycle) or servers (highest server count).",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tag",
				Description: "Only bots with this tag (case-insensitive).",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "certified",
				Description: "true to only return certified bots.",
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

	query := r.URL.Query()

	sort := query.Get("sort")

	if sort == "" {
		sort = "new"
	}

	if !sorts[sort] {
		return resp.BadRequest("sort must be new, trending, votes or servers")
	}

	var tag pgtype.Text

	if raw := strings.TrimSpace(query.Get("tag")); raw != "" {
		if len(raw) > maxTagLength {
			return resp.BadRequest("tag is too long")
		}

		tag = pgtype.Text{String: raw, Valid: true}
	}

	certifiedOnly := query.Get("certified") == "true"

	q := db.New(state.Pool)

	rows, err := q.BrowseBots(d.Context, db.BrowseBotsParams{
		Sort:          sort,
		CertifiedOnly: certifiedOnly,
		Tag:           tag,
		Lim:           perPage,
		Off:           int32((pageNum - 1) * perPage),
	})

	if err != nil {
		return resp.Err("Error while getting bots [db fetch]", err)
	}

	results := make([]types.IndexBot, len(rows))

	for i, row := range rows {
		results[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}

	if err := assets.ResolveIndexBots(d.Context, results); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving bots.", err)
	}

	count, err := q.CountBrowseBots(d.Context, db.CountBrowseBotsParams{
		Sort:          sort,
		CertifiedOnly: certifiedOnly,
		Tag:           tag,
	})

	if err != nil {
		return resp.Err("Error while getting bot count", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.IndexBot]{
			Count:   uint64(count),
			Results: results,
			PerPage: perPage,
		},
	}
}
