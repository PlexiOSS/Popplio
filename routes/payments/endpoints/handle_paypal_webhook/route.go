// Copyright (C) 2026 NodeByte LTD

package handle_paypal_webhook

import (
	"io"
	"net/http"

	"popplio/api/resp"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"

	"github.com/plutov/paypal/v4"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Handle PayPal Webhook",
		Description: "Handles PayPal refund and dispute webhooks. Not intended for public use; every event is verified with PayPal before it is acted on.",
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if state.Paypal == nil || state.PaypalWebhookID == "" {
		return resp.Status(http.StatusFailedDependency, "PayPal webhooks are not configured yet! Please try again in a few moments?")
	}

	verified, err := state.Paypal.VerifyWebhookSignature(d.Context, r, state.PaypalWebhookID)

	if err != nil {
		return resp.Err("Failed to verify PayPal webhook", err)
	}

	if verified.VerificationStatus != "SUCCESS" {
		state.Logger.Error("PayPal webhook failed verification", zap.String("status", verified.VerificationStatus))
		return resp.BadRequest("Invalid webhook signature")
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		return resp.BadRequest("Invalid request body")
	}

	var event paypal.AnyEvent

	if err := jsonimpl.Unmarshal(body, &event); err != nil {
		return resp.BadRequest("Invalid request body")
	}

	return assets.HandlePaypalEvent(d.Context, event)
}
