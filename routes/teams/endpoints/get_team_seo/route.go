package get_team_seo

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/google/uuid"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team SEO Info",
		Description: "Gets the minimal SEO information about a team for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The team ID, name or vanity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	tid := chi.URLParam(r, "id")

	if _, err := uuid.Parse(tid); err != nil {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	row, err := db.New(state.Pool).GetTeamSEO(d.Context, tid)

	if err != nil {
		return resp.Err("Error getting team SEO info [db queryrow]", err, zap.String("id", tid))
	}

	seoData := types.SEO{
		ID:   row.ID,
		Name: row.Name,
		Short: func() string {
			if !row.Short.Valid || row.Short.String == "" {
				return "View the team " + row.Name + " on Omniplex"
			}

			return row.Short.String
		}(),
	}

	return uapi.HttpResponse{
		Json: seoData,
	}
}
