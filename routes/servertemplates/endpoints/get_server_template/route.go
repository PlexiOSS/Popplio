// Package get_server_template implements GET /server-templates/{id} — "Get
// Server Template".
//
// Gets a single server template by its internal ID
package get_server_template

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/servertemplates/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server Template",
		Description: "Gets a single server template by its internal ID",
		Resp:        types.ServerTemplate{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The template's internal ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	row, err := db.New(state.Pool).GetServerTemplateByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting server template [db fetch]", err)
	}

	tmpl := types.ServerTemplate{
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

	if err := assets.ResolveServerTemplate(d.Context, &tmpl); err != nil {
		return resp.ErrDetail("Error resolving server template", err)
	}

	return uapi.HttpResponse{
		Json: tmpl,
	}
}
