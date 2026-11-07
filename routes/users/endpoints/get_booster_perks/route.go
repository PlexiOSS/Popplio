// Copyright (C) 2026 NodeByte LTD

package get_booster_perks

import (
	"net/http"

	"popplio/api/resp"
	"popplio/routes/payments/assets"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Booster Perks",
		Description: "Returns whether the user is boosting the main Omniplex Discord server, the listing their monthly booster premium goes to and when it next renews. Only the user themselves can check.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BoosterPerks{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	out, err := assets.BoosterPerksFor(d.Context, chi.URLParam(r, "id"))

	if err != nil {
		return resp.Err("Failed to load booster perks", err)
	}

	return uapi.HttpResponse{Json: out}
}
