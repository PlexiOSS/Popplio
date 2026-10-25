package handle_stripe_webhook

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"popplio/api/resp"
	"popplio/notifications"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/webhook"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Handle Stripe Webhook",
		Description: "Handles stripe payment webhooks. Not intended for public use and firewalled to only stripe IPs.",
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if state.StripeWebhSecret == "" {
		return resp.Status(http.StatusFailedDependency, "Stripe webhooks are not configured yet! Please try again in a few moments?")
	}

	ip := requestIP(r)

	if !slices.Contains(state.StripeWebhIPList, ip) {
		state.Logger.Error("IP is not allowed to access this endpoint", zap.String("ip", ip), zap.String("remoteAddr", r.RemoteAddr))
		return resp.Forbidden("IP is not allowed to access this endpoint")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		state.Logger.Error("Failed to read request body", zap.Error(err))
		return resp.BadRequest("Invalid request body")
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), state.StripeWebhSecret)

	if err != nil {
		state.Logger.Error("Failed to construct event", zap.Error(err))
		return resp.BadRequest("Invalid request body")
	}

	var s stripe.CheckoutSession
	var failed bool

	switch event.Type {
	case "checkout.session.completed":
		err := jsonimpl.Unmarshal(event.Data.Raw, &s)
		if err != nil {
			return resp.Err("Failed to unmarshal event data", err)
		}

		if s.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			state.Logger.Error("Payment status is not paid")
			return uapi.HttpResponse{
				Status: http.StatusOK,
				Data:   "Payment status is not paid yet!",
			}
		}

	case "checkout.session.async_payment_succeeded":
		err := jsonimpl.Unmarshal(event.Data.Raw, &s)
		if err != nil {
			return resp.Err("Failed to unmarshal event data", err)
		}

	case "checkout.session.async_payment_failed":
		var s stripe.CheckoutSession
		err := jsonimpl.Unmarshal(event.Data.Raw, &s)
		if err != nil {
			return resp.Err("Failed to unmarshal event data", err)
		}

		failed = true

	default:
		state.Logger.Error("Unknown event type", zap.String("event", string(event.Type)))
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Unknown event type: " + string(event.Type),
		}
	}

	var payload assets.PerkData

	err = jsonimpl.Unmarshal([]byte(s.ClientReferenceID), &payload)

	if err != nil {
		state.Logger.Error("Failed to unmarshal client reference id", zap.Error(err))
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Failed to unmarshal client reference id: " + err.Error(),
		}
	}

	if failed {
		state.Logger.Error("Payment failed for user " + payload.UserID)

		go notifications.PushNotification(payload.UserID, types.Alert{
			Title:    "Payment Failed",
			Message:  "Your payment for \"" + payload.ProductName + "\" for " + payload.For + " has failed. Please contact our support team to learn more!",
			Type:     types.AlertTypeError,
			Priority: types.AlertPriorityHigh,
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/support", Valid: true},
			Category: types.AlertCategoryPayments,
		})

		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Payment failed for user " + payload.UserID + " for product " + payload.ProductID + "( " + payload.ProductName + " )" + " for " + payload.For,
		}
	}

	fulfillCtx, cancel := context.WithTimeout(state.Context, 25*time.Second)
	defer cancel()

	fulfilled, err := assets.Fulfill(fulfillCtx, "stripe", s.ID, payload)

	if errors.Is(err, assets.ErrInvalidPurchase) {
		state.Logger.Error("Stripe purchase could not be applied", zap.Error(err), zap.String("session", s.ID), zap.Any("payload", payload))

		go notifications.PushNotification(payload.UserID, types.Alert{
			Title:    "Perk Delivery Failed",
			Message:  "Your payment for \"" + payload.ProductName + "\" for " + payload.For + " went through but couldn't be applied. Please contact our support team so we can sort it out: " + err.Error(),
			Type:     types.AlertTypeError,
			Priority: types.AlertPriorityHigh,
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/support", Valid: true},
			Category: types.AlertCategoryPayments,
		})

		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Purchase could not be applied: " + err.Error(),
		}
	}

	if err != nil {
		state.Logger.Error("Failed to fulfill Stripe purchase, Stripe will retry", zap.Error(err), zap.String("session", s.ID), zap.Any("payload", payload))
		return resp.Status(http.StatusInternalServerError, "Failed to fulfill purchase, please retry")
	}

	if !fulfilled {
		state.Logger.Info("Stripe purchase already fulfilled", zap.String("session", s.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}

func requestIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}
