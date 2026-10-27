package create_auto_vote_portal

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/db"
	"popplio/routes/payments/assets"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Manage Auto-vote Subscription",
		Description: "Returns a Stripe billing portal URL where the user can update their card or cancel their auto-vote subscription.",
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
	sub, err := db.New(state.Pool).GetAutoVoteSubscription(d.Context, chi.URLParam(r, "id"))

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.BadRequest("You don't have an auto-vote subscription")
	}

	if err != nil {
		return resp.Err("Failed to load auto-vote subscription", err)
	}

	url, err := autovote.PortalURL(sub.StripeCustomerID)

	if err != nil {
		return resp.Err("Failed to open the billing portal", err)
	}

	return uapi.HttpResponse{Json: assets.RedirectUser{URL: url}}
}
