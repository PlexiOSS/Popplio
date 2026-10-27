package create_auto_vote_checkout

import (
	"errors"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/db"
	"popplio/routes/payments/assets"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Start Auto-vote Subscription",
		Description: "Creates a Stripe Checkout session for the monthly auto-vote subscription and returns its URL.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: assets.RedirectUser{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      time.Minute,
		MaxRequests: 3,
		Bucket:      "payments",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err)
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	userID := chi.URLParam(r, "id")

	sub, err := db.New(state.Pool).GetAutoVoteSubscription(d.Context, userID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resp.Err("Failed to check auto-vote subscription", err)
	}

	if err == nil && autovote.Active(sub.Status, sub.CurrentPeriodEnd) {
		return resp.BadRequest("You already have an active auto-vote subscription")
	}

	url, err := autovote.CheckoutURL(userID)

	if err != nil {
		state.Logger.Error("Failed to create auto-vote checkout", zap.Error(err), zap.String("userId", userID))
		return resp.Err("Failed to start checkout", err)
	}

	return uapi.HttpResponse{Json: assets.RedirectUser{URL: url}}
}
