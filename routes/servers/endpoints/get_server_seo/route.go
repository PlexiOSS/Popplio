package get_server_seo

import (
	"errors"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server SEO Info",
		Description: "Gets the minimal SEO information about a server for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	row, err := db.New(state.Pool).GetServerNameAndShort(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting server [queryrow]", err, zap.String("serverID", id))
	}

	seoData := types.SEO{
		ID:    id,
		Name:  row.Name,
		Short: row.Short,
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
