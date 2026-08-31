// Package get_all_server_templates implements GET /server-templates/@all —
// "Get All Server Templates".
//
// Gets all server templates on the list, paginated, optionally filtered by
// tag
package get_all_server_templates

import (
	"net/http"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/routes/servertemplates/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 12

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Server Templates",
		Description: "Gets all server templates on the list, paginated, optionally filtered by tag",
		Resp:        types.PagedResult[[]types.ServerTemplate]{},
		RespName:    "PagedResultServerTemplate",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tags",
				Description: "Comma-separated tags to filter by (matches any). Omit to return every template.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "owner",
				Description: "Filter to templates submitted by this user ID. Omit to return every submitter's.",
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

	tags := []string{}

	if raw := r.URL.Query().Get("tags"); raw != "" {
		for _, tag := range strings.Split(raw, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	owner := r.URL.Query().Get("owner")

	q := db.New(state.Pool)

	rows, err := q.GetServerTemplatesPaged(d.Context, db.GetServerTemplatesPagedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
		Tags:   tags,
		Owner:  owner,
	})

	if err != nil {
		return resp.Err("Error while querying server templates [db fetch]", err)
	}

	templates := make([]types.ServerTemplate, len(rows))
	for i, row := range rows {
		templates[i] = types.ServerTemplate{
			ID:         row.ID,
			Code:       row.Code,
			Name:       row.Name,
			Short:      row.Short,
			Tags:       row.Tags,
			NSFW:       row.Nsfw,
			OwnerID:    row.Owner,
			UsageCount: int(row.UsageCount),
			CreatedAt:  row.CreatedAt.Time,
			UpdatedAt:  row.UpdatedAt.Time,
		}

		if err := assets.ResolveServerTemplate(d.Context, &templates[i]); err != nil {
			return resp.ErrDetail("Error resolving server template", err, zap.String("id", templates[i].ID))
		}
	}

	count, err := q.CountServerTemplatesFiltered(d.Context, db.CountServerTemplatesFilteredParams{
		Tags:  tags,
		Owner: owner,
	})

	if err != nil {
		return resp.Err("Error while querying server templates [db count]", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.ServerTemplate]{
			Count:   uint64(count),
			PerPage: perPage,
			Results: templates,
		},
	}
}
