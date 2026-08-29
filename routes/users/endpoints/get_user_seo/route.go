// Package get_user_seo implements GET /users/{id}/seo — "Get User SEO Info".
//
// Gets a users SEO data by id
package get_user_seo

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User SEO Info",
		Description: "Gets a users SEO data by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.SEO{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	name := chi.URLParam(r, "id")

	row, err := db.New(state.Pool).GetUserAboutAndID(d.Context, name)

	if err != nil {
		state.Logger.Error("Failed to get user seo", zap.Error(err), zap.String("user_id", name))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	user, err := dovewing.GetUser(d.Context, row.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Failed to get user seo", err, zap.String("user_id", name))
	}

	seo := types.SEO{
		ID:     user.ID,
		Name:   user.DisplayName,
		Avatar: user.Avatar,
		Short:  row.About.String,
	}

	return uapi.HttpResponse{
		Json: seo,
	}
}
