// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"popplio/audit"
	"popplio/state"

	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/plutov/paypal/v4"
)

type paypalResource struct {
	ID                   string        `json:"id"`
	Status               string        `json:"status"`
	Links                []paypal.Link `json:"links"`
	DisputedTransactions []struct {
		SellerTransactionID string `json:"seller_transaction_id"`
	} `json:"disputed_transactions"`
}

type paypalCapture struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	SupplementaryData struct {
		RelatedIDs struct {
			OrderID string `json:"order_id"`
		} `json:"related_ids"`
	} `json:"supplementary_data"`
}

var revokePaypalPurchase = RevokePurchase

func paypalReason(eventType string) string {
	switch eventType {
	case "PAYMENT.CAPTURE.REFUNDED":
		return RevokeRefunded
	case "PAYMENT.CAPTURE.REVERSED", "CUSTOMER.DISPUTE.CREATED":
		return RevokeDisputed
	default:
		return ""
	}
}

func paypalCaptureIDs(resourceType string, res paypalResource) []string {
	var ids []string

	add := func(id string) {
		if id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}

	for _, t := range res.DisputedTransactions {
		add(t.SellerTransactionID)
	}

	for _, l := range res.Links {
		if l.Rel != "up" {
			continue
		}

		if _, after, ok := strings.Cut(l.Href, "/payments/captures/"); ok {
			add(strings.Trim(after, "/"))
		}
	}

	if len(ids) == 0 && strings.EqualFold(resourceType, "capture") {
		add(res.ID)
	}

	return ids
}

func getPaypalCapture(ctx context.Context, id string) (*paypalCapture, error) {
	req, err := state.Paypal.NewRequest(ctx, http.MethodGet, state.Paypal.APIBase+"/v2/payments/captures/"+url.PathEscape(id), nil)

	if err != nil {
		return nil, err
	}

	var c paypalCapture

	if err := state.Paypal.SendWithAuth(req, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func HandlePaypalEvent(ctx context.Context, event paypal.AnyEvent) uapi.HttpResponse {
	reason := paypalReason(event.EventType)

	if reason == "" {
		return webhookOK("event ignored")
	}

	record := func(action string, extra map[string]any) {
		data := map[string]any{"event": event.ID, "event_type": event.EventType}

		for k, v := range extra {
			data[k] = v
		}

		audit.Record(audit.Event{Source: audit.SourcePayPal, ActorType: audit.ActorSystem, Action: action, Data: data})
	}

	var res paypalResource

	if err := jsonimpl.Unmarshal(event.Resource, &res); err != nil {
		record("payment."+reason+"_unmatched", map[string]any{"error": "unreadable resource"})
		return webhookOK("unreadable resource")
	}

	ids := paypalCaptureIDs(event.ResourceType, res)

	if len(ids) == 0 {
		record("payment."+reason+"_unmatched", map[string]any{"resource": res.ID})
		return webhookOK("no capture in event")
	}

	for _, id := range ids {
		capture, err := getPaypalCapture(ctx, id)

		if err != nil {
			return webhookRetry("fetching paypal capture", err)
		}

		info := map[string]any{"capture": id, "capture_status": capture.Status}

		if reason == RevokeRefunded && capture.Status != "REFUNDED" {
			record("payment.partially_refunded", info)
			continue
		}

		orderID := capture.SupplementaryData.RelatedIDs.OrderID

		if orderID == "" {
			record("payment."+reason+"_unmatched", info)
			continue
		}

		if _, err := revokePaypalPurchase(ctx, "paypal", orderID, reason, reason == RevokeRefunded); err != nil {
			if errors.Is(err, ErrNotFulfilled) {
				info["order"] = orderID
				record("payment."+reason+"_unmatched", info)
				continue
			}

			return webhookRetry("revoking paypal purchase", err)
		}
	}

	return webhookOK("paypal " + reason + " processed")
}
