package create_auto_vote_checkout

import (
	"errors"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/db"
	"popplio/promo"
	"popplio/routes/payments/assets"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"go.uber.org/zap"
)

type CreateAutoVoteCheckout struct {
	PromoCode string `json:"promo_code" validate:"omitempty,max=40" msg:"promo_code is too long."`
}

var compiledMessages = uapi.CompileValidationErrors(CreateAutoVoteCheckout{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Start Auto-vote Subscription",
		Description: "Creates a Stripe Checkout session for the monthly auto-vote subscription and returns its URL. The body is optional and can carry a promo code.",
		Req:         CreateAutoVoteCheckout{},
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

	var body CreateAutoVoteCheckout

	if r.ContentLength != 0 {
		if hresp, ok := uapi.MarshalReqWithHeaders(r, &body, limit.Headers()); !ok {
			return hresp
		}

		if err := state.Validator.Struct(body); err != nil {
			return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
		}
	}

	sub, err := db.New(state.Pool).GetAutoVoteSubscription(d.Context, userID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resp.Err("Failed to check auto-vote subscription", err)
	}

	if err == nil && autovote.Active(sub.Status, sub.CurrentPeriodEnd) {
		return resp.BadRequest("You already have an active auto-vote subscription")
	}

	var promoCode *promo.Code
	var discounts []*stripe.CheckoutSessionDiscountParams

	if body.PromoCode != "" {
		c, _, err := promo.Validate(d.Context, userID, body.PromoCode, promo.ProductAutoVote, autovote.PriceCents)

		if err != nil {
			return promo.ErrResponse("Error while checking promo code", err)
		}

		if err := promo.ReleasePending(d.Context, c, userID); err != nil {
			return promo.ErrResponse("Error while checking promo code", err)
		}

		promoCode = c
		discounts = promo.SessionDiscount(c)
	}

	s, err := autovote.Checkout(userID, discounts)

	if err != nil {
		state.Logger.Error("Failed to create auto-vote checkout", zap.Error(err), zap.String("userId", userID))

		if promoCode != nil {
			return promo.StripeErrResponse("Failed to start checkout", err)
		}

		return resp.Err("Failed to start checkout", err)
	}

	if promoCode != nil {
		if err := promo.Record(d.Context, promoCode, userID, promo.ProductAutoVote, s.ID); err != nil {
			if _, expireErr := session.Expire(s.ID, nil); expireErr != nil {
				state.Logger.Error("Failed to expire Stripe session after a failed promo code record", zap.Error(expireErr), zap.String("session", s.ID))
			}

			return resp.Err("Error while applying promo code", err)
		}
	}

	return uapi.HttpResponse{Json: assets.RedirectUser{URL: s.URL}}
}
