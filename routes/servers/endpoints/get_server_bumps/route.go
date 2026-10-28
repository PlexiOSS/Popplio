package get_server_bumps

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/bumps"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server Bumps",
		Description: "Returns when a server was last bumped with /bump, when it can next be bumped, how many bumps it got in the last 30 days and its top bumpers.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The server's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ServerBumpInfo{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	info, err := bumps.Info(d.Context, chi.URLParam(r, "id"), 5)

	if errors.Is(err, bumps.ErrNotFound) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Failed to load server bumps", err)
	}

	return uapi.HttpResponse{Json: info}
}
