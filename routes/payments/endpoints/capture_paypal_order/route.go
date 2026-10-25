// Package capture_paypal_order implements GET
// /payments/paypal/capture/{ref_id} — "Capture Paypal Order".
//
// Captures a paypal order, giving any needed perks. This is an internal
// endpoint.
package capture_paypal_order

import (
	"context"
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/routes/payments/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/plutov/paypal/v4"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Capture Paypal Order",
		Description: "Captures a paypal order, giving any needed perks. This is an internal endpoint.",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "ref_id",
				Description: "The reference ID of the order returned during paypals redirect back to us",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if state.Paypal == nil {
		return resp.Status(http.StatusServiceUnavailable, "Paypal is currently not available as a payment option. Please contact support!")
	}

	refId := chi.URLParam(r, "ref_id")

	if refId == "" {
		return resp.BadRequest("Missing ref_id")
	}

	// Get order ID from redis
	orderIdRedis := state.Redis.Get(d.Context, "paypal:"+refId)

	if orderIdRedis.Err() != nil {
		return resp.BadRequest("Invalid ref_id. Please contact support if you believe this is an error.")
	}

	orderId := orderIdRedis.Val()

	captured, err := state.Paypal.CaptureOrder(d.Context, orderId, paypal.CaptureOrderRequest{})

	if err != nil {
		return resp.ErrDetail("Failed to capture paypal order", err, zap.String("order_id", orderId))
	}

	if captured.Status == "VOIDED" {
		return resp.BadRequest("Order is voided. Please contact support if you believe this is an error.")
	}

	if len(captured.PurchaseUnits) == 0 {
		// Refund the order
		err = refundCapture(d.Context, captured)

		if err != nil {
			return resp.ErrBody("Failed to refund order [len(captured.PurchaseUnits) == 0]", "Failed to refund order.", err, zap.String("order_id", orderId))
		}

		return resp.BadRequest("No purchase units found. Please contact support if you believe this is an error.")
	}

	if len(captured.PurchaseUnits[0].Items) == 0 {
		// Refund the order
		err = refundCapture(d.Context, captured)

		if err != nil {
			return resp.ErrBody("Failed to refund order [captured.PurchaseUnits[0].Items == 0]", "Failed to refund order.", err, zap.String("order_id", orderId))
		}

		return resp.BadRequest("No purchase items found. Please contact support if you believe this is an error.")
	}

	var productJson = captured.PurchaseUnits[0].Items[0].SKU

	var product assets.PerkData

	err = jsonimpl.Unmarshal([]byte(productJson), &product)

	if err != nil {
		// Refund the order
		err = refundCapture(d.Context, captured)

		if err != nil {
			return resp.ErrBody("Failed to refund order [jsonimpl.Unmarshal]", "Failed to refund order.", err, zap.String("order_id", orderId))
		}

		return resp.Err("Failed to unmarshal product json", err, zap.String("json", productJson))
	}

	if _, fulfillErr := assets.Fulfill(d.Context, "paypal", orderId, product); fulfillErr != nil {
		state.Logger.Error("Failed to give perks, refunding", zap.Error(fulfillErr), zap.String("order_id", orderId), zap.String("json", productJson))

		if err := refundCapture(d.Context, captured); err != nil {
			return resp.ErrBody("Failed to refund order [Fulfill: err != nil]", "Failed to refund order.", err, zap.String("order_id", orderId))
		}

		return resp.BadRequest("Failed to give perks. Your payment has been refunded.")
	}

	state.Redis.Del(d.Context, "paypal:"+refId)

	return uapi.HttpResponse{
		Redirect: state.Config.Sites.Frontend + "/payments/success",
	}
}

func refundCapture(ctx context.Context, captured *paypal.CaptureOrderResponse) error {
	for _, unit := range captured.PurchaseUnits {
		if unit.Payments == nil {
			continue
		}

		for _, capture := range unit.Payments.Captures {
			if capture.ID == "" {
				continue
			}

			_, err := state.Paypal.RefundCapture(ctx, capture.ID, paypal.RefundCaptureRequest{})
			return err
		}
	}

	return errors.New("paypal order has no capture to refund")
}
