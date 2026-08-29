package get_pack_seo

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
		Summary:     "Get Pack SEO Info",
		Description: "Gets the minimal SEO information about a pack for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The packs ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	row, err := db.New(state.Pool).GetPackSEO(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Failed to get pack seo", err, zap.String("url", id))
	}

	seoData := types.SEO{
		ID:     id,
		Name:   row.Name,
		Avatar: "",
		Short:  row.Short,
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
