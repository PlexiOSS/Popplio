// Copyright (C) 2026 NodeByte LTD

package put_booster_perks

import (
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/bgtasks"
	"popplio/db"
	"popplio/listing"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

var compiledMessages = uapi.CompileValidationErrors(types.BoosterPerksUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Set Booster Perks",
		Description: "Picks the listing the user's monthly booster premium goes to, or clears it with null target_type and target_id. Only boosters can pick a listing. If a month is due, it's applied straight away and just_applied is true.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.BoosterPerksUpdate{},
		Resp: types.BoosterPerks{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      time.Minute,
		MaxRequests: 5,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err)
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	userID := chi.URLParam(r, "id")

	var payload types.BoosterPerksUpdate

	if hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers()); !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	clearing := payload.TargetType == nil || payload.TargetID == nil || *payload.TargetID == ""

	if (payload.TargetType == nil) != (payload.TargetID == nil) {
		return resp.BadRequest("Send both target_type and target_id, or neither to clear the listing")
	}

	_, boosting := assets.LiveBoosting(userID)

	if !clearing && !boosting {
		return resp.BadRequest("Boost the Omniplex Discord server to get booster perks")
	}

	params := db.SetBoosterTargetParams{UserID: userID, Boosting: boosting}

	if !clearing {
		public, err := listing.EntityIsPublic(d.Context, *payload.TargetType, *payload.TargetID)

		if err != nil || !public {
			return resp.BadRequest("Only approved bots and servers can get premium")
		}

		params.TargetType = pgtype.Text{String: *payload.TargetType, Valid: true}
		params.TargetID = pgtype.Text{String: *payload.TargetID, Valid: true}
	}

	if err := db.New(state.Pool).SetBoosterTarget(d.Context, params); err != nil {
		return resp.Err("Failed to save booster perks", err)
	}

	applied := false

	if !clearing {
		out, err := assets.BoosterPerksFor(d.Context, userID)

		if err == nil && out.NextRenewalAt != nil && !out.NextRenewalAt.After(time.Now()) {
			applied = bgtasks.RenewBoosterPremium(d.Context, userID, *payload.TargetType, *payload.TargetID)
		}
	}

	out, err := assets.BoosterPerksFor(d.Context, userID)

	if err != nil {
		return resp.Err("Failed to load booster perks", err)
	}

	out.JustApplied = applied

	return uapi.HttpResponse{Json: out}
}
