// Package check_promo_code implements POST /users/{id}/promo-codes/check —
// "Check Promo Code".
//
// Checks whether a promo code can be used by the user for premium or
// Auto-vote and previews the discount it gives.
package check_promo_code

import (
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/promo"
	"popplio/routes/payments/assets"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type CheckPromoCode struct {
	Code    string `json:"code" validate:"required,max=40" msg:"Enter a promo code."`
	Product string `json:"product" validate:"required,oneof=premium autovote" msg:"product must be 'premium' or 'autovote'."`
	Plan    string `json:"plan" validate:"required_if=Product premium" msg:"plan is required for premium."`
	Credits int    `json:"credits" validate:"omitempty,min=0" msg:"credits can't be negative."`
}

var compiledMessages = uapi.CompileValidationErrors(CheckPromoCode{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Check Promo Code",
		Description: "Checks whether a promo code can be used for premium or Auto-vote and previews the discount. For premium, credits are applied before the code.",
		Req:         CheckPromoCode{},
		Resp:        promo.Preview{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      time.Minute,
		MaxRequests: 10,
		Bucket:      "promo_code_check",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err)
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var body CheckPromoCode

	hresp, ok := uapi.MarshalReqWithHeaders(r, &body, limit.Headers())

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(body); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	subtotal := autovote.PriceCents

	if body.Product == promo.ProductPremium {
		subtotal = 0

		for _, plan := range assets.Plans {
			if plan.ID == body.Plan {
				priceCents := assets.PriceCents(plan.Price)
				_, creditCents := assets.CreditDiscount(priceCents, body.Credits)
				subtotal = priceCents - creditCents
				break
			}
		}

		if subtotal == 0 {
			return resp.BadRequest("Unknown premium plan")
		}
	}

	_, preview, err := promo.Validate(d.Context, chi.URLParam(r, "id"), body.Code, body.Product, subtotal)

	if err != nil {
		return promo.ErrResponse("Error while checking promo code", err)
	}

	return uapi.HttpResponse{Json: preview, Headers: limit.Headers()}
}
