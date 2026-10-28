package get_all_servers

import (
	"net/http"
	"strings"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/routes/servers/assets"
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
	"bumped":   true,
	"members":  true,
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Servers",
		Description: "Gets approved and certified, publicly listed servers, filtered and sorted. Returns a set of paginated ``IndexServer`` objects",
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
				Description: "Sort order: new (default; newest first, with boosted listings on top), trending (net votes in the last 7 days, only servers with at least one vote in that window), votes (most votes this cycle) members (highest member count) or bumped (most recently bumped with /bump first).",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tag",
				Description: "Only servers with this tag (case-insensitive).",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "certified",
				Description: "true to only return certified servers.",
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
		return resp.BadRequest("sort must be new, trending, votes, members or bumped")
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

	rows, err := q.BrowseServers(d.Context, db.BrowseServersParams{
		Sort:          sort,
		CertifiedOnly: certifiedOnly,
		Tag:           tag,
		Lim:           perPage,
		Off:           int32((pageNum - 1) * perPage),
	})

	if err != nil {
		return resp.Err("Error while getting servers [db fetch]", err)
	}

	results := make([]types.IndexServer, len(rows))

	for i, row := range rows {
		results[i] = types.IndexServer{
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

	if err := assets.ResolveIndexServers(d.Context, results); err != nil {
		return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving servers.", err)
	}

	count, err := q.CountBrowseServers(d.Context, db.CountBrowseServersParams{
		Sort:          sort,
		CertifiedOnly: certifiedOnly,
		Tag:           tag,
	})

	if err != nil {
		return resp.Err("Error while getting server count", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.IndexServer]{
			Count:   uint64(count),
			Results: results,
			PerPage: perPage,
		},
	}
}
