// Copyright (C) 2026 NodeByte LTD

package put_site_prefs

import (
	"encoding/json"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

const MaxBytes = 8192

var compiledMessages = uapi.CompileValidationErrors(types.SitePrefsUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Set Site Preferences",
		Description: "Saves the user's website customization settings, replacing what was there. prefs must be a JSON object of at most 8 KB. Only the user themselves can save them.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.SitePrefsUpdate{},
		Resp: types.SitePrefs{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      time.Minute,
		MaxRequests: 30,
		Bucket:      "site_prefs",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err)
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var payload types.SitePrefsUpdate

	if hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers()); !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	if raw, err := json.Marshal(payload.Prefs); err != nil || len(raw) > MaxBytes {
		return resp.BadRequest("prefs must be a JSON object of at most 8 KB")
	}

	updated, err := db.New(state.Pool).UpsertSitePrefs(d.Context, db.UpsertSitePrefsParams{
		UserID: chi.URLParam(r, "id"),
		Prefs:  payload.Prefs,
	})

	if err != nil {
		return resp.Err("Failed to save site preferences", err)
	}

	t := updated.Time

	return uapi.HttpResponse{Json: types.SitePrefs{Prefs: payload.Prefs, UpdatedAt: &t}}
}
