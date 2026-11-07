// Copyright (C) 2026 NodeByte LTD

package put_auto_vote_slots

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/stripe/stripe-go/v75"
	"go.uber.org/zap"
)

var compiledMessages = uapi.CompileValidationErrors(types.AutoVoteSlotsUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Set Auto-vote Extra Slots",
		Description: "Sets how many extra listing slots the user's Auto-vote subscription has, on top of the 3 included. Adding slots charges the prorated difference straight away and fails if the payment does. Removing slots credits the difference on the next invoice. Returns 204 on success.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.AutoVoteSlotsUpdate{},
		Resp: types.ApiError{},
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

	var payload types.AutoVoteSlotsUpdate

	if hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers()); !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	inUse, err := autovote.SetExtraSlots(d.Context, userID, payload.ExtraSlots)

	switch {
	case err == nil:
		return uapi.DefaultResponse(http.StatusNoContent)
	case errors.Is(err, autovote.ErrSlotsInactive):
		return resp.BadRequest("You need an active Auto-vote subscription to add slots")
	case errors.Is(err, autovote.ErrSlotsInUse):
		return resp.BadRequest("You have " + strconv.Itoa(inUse) + " listings set up. Remove some before dropping to " + strconv.Itoa(autovote.Limit(payload.ExtraSlots)+autovote.BoosterBonus(d.Context, userID)) + " slots.")
	}

	var stripeErr *stripe.Error

	if errors.As(err, &stripeErr) && stripeErr.Type == stripe.ErrorTypeCard {
		return resp.BadRequest("Your card was declined, so no slots were added. Update your card in billing and try again.")
	}

	state.Logger.Error("Failed to change auto-vote slots", zap.Error(err), zap.String("userId", userID))

	return resp.Err("Failed to change your slots", err)
}
