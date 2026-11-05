package assets

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"popplio/state"
	"popplio/testdb"

	"github.com/plutov/paypal/v4"
)

func TestPaypalCaptureIDs(t *testing.T) {
	refund := paypalResource{
		ID: "REF1",
		Links: []paypal.Link{
			{Rel: "self", Href: "https://api.paypal.com/v2/payments/refunds/REF1"},
			{Rel: "up", Href: "https://api.paypal.com/v2/payments/captures/CAP1"},
		},
	}

	if got := paypalCaptureIDs("refund", refund); !slices.Equal(got, []string{"CAP1"}) {
		t.Errorf("refund: got %v", got)
	}

	dispute := paypalResource{ID: "PP-D-1"}
	dispute.DisputedTransactions = append(dispute.DisputedTransactions,
		struct {
			SellerTransactionID string `json:"seller_transaction_id"`
		}{"CAP2"},
		struct {
			SellerTransactionID string `json:"seller_transaction_id"`
		}{"CAP2"},
	)

	if got := paypalCaptureIDs("dispute", dispute); !slices.Equal(got, []string{"CAP2"}) {
		t.Errorf("dispute: got %v", got)
	}

	if got := paypalCaptureIDs("capture", paypalResource{ID: "CAP3"}); !slices.Equal(got, []string{"CAP3"}) {
		t.Errorf("capture: got %v", got)
	}

	if got := paypalCaptureIDs("refund", paypalResource{ID: "REF2"}); len(got) != 0 {
		t.Errorf("refund without links: got %v", got)
	}
}

func fakePaypal(t *testing.T, captures map[string]string) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/v1/oauth2/token" {
			io.WriteString(w, `{"access_token":"test","token_type":"Bearer","expires_in":3600}`)
			return
		}

		if body, ok := captures[strings.TrimPrefix(r.URL.Path, "/v2/payments/captures/")]; ok {
			io.WriteString(w, body)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"name":"RESOURCE_NOT_FOUND"}`)
	}))
	t.Cleanup(srv.Close)

	c, err := paypal.NewClient("id", "secret", srv.URL)

	if err != nil {
		t.Fatal(err)
	}

	prevClient, prevRevoke := state.Paypal, revokePaypalPurchase
	state.Paypal = c
	revokePaypalPurchase = revoke

	t.Cleanup(func() {
		state.Paypal, revokePaypalPurchase = prevClient, prevRevoke
	})
}

func seedPaypalPurchase(t *testing.T, orderID, userID, botID string, fulfilledAt time.Time) {
	t.Helper()

	testdb.Exec(t, `INSERT INTO payment_fulfillments (provider, external_id, user_id, product_id, for_type, for_id, fulfilled_at, premium_hours)
		VALUES ('paypal', $1, $2, 'premium', 'bot', $3, $4, 720)`, orderID, userID, botID, fulfilledAt)
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE external_id = $1", orderID)
}

func paypalEvent(eventType, resourceType, resource string) paypal.AnyEvent {
	var e paypal.AnyEvent
	e.ID = "WH-" + eventType
	e.EventType = eventType
	e.ResourceType = resourceType
	e.Resource = []byte(resource)
	return e
}

func capture(id, status, orderID string) string {
	return `{"id":"` + id + `","status":"` + status + `","supplementary_data":{"related_ids":{"order_id":"` + orderID + `"}}}`
}

func TestPaypalRefundRevokesPremium(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720)
	order, capID := "ORDER-"+botID, "CAP-"+botID
	seedPaypalPurchase(t, order, userID, botID, granted)
	fakePaypal(t, map[string]string{capID: capture(capID, "REFUNDED", order)})

	res := HandlePaypalEvent(context.Background(), paypalEvent("PAYMENT.CAPTURE.REFUNDED", "refund",
		`{"id":"REF-1","links":[{"rel":"up","href":"https://api.paypal.com/v2/payments/captures/`+capID+`"}]}`))

	if res.Status != http.StatusOK {
		t.Fatalf("status = %d (%v)", res.Status, res.Data)
	}

	if premiumOf(t, botID) {
		t.Error("bot still premium after a full PayPal refund")
	}
}

func TestPaypalPartialRefundKeepsPremium(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720)
	order, capID := "ORDER-"+botID, "CAP-"+botID
	seedPaypalPurchase(t, order, userID, botID, granted)
	fakePaypal(t, map[string]string{capID: capture(capID, "PARTIALLY_REFUNDED", order)})

	res := HandlePaypalEvent(context.Background(), paypalEvent("PAYMENT.CAPTURE.REFUNDED", "refund",
		`{"id":"REF-2","links":[{"rel":"up","href":"https://api.paypal.com/v2/payments/captures/`+capID+`"}]}`))

	if res.Status != http.StatusOK {
		t.Fatalf("status = %d (%v)", res.Status, res.Data)
	}

	if !premiumOf(t, botID) {
		t.Error("partial refund removed premium")
	}
}

func TestPaypalDisputeRevokesPremium(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720)
	order, capID := "ORDER-"+botID, "CAP-"+botID
	seedPaypalPurchase(t, order, userID, botID, granted)
	fakePaypal(t, map[string]string{capID: capture(capID, "COMPLETED", order)})

	res := HandlePaypalEvent(context.Background(), paypalEvent("CUSTOMER.DISPUTE.CREATED", "dispute",
		`{"dispute_id":"PP-D-1","disputed_transactions":[{"seller_transaction_id":"`+capID+`"}]}`))

	if res.Status != http.StatusOK {
		t.Fatalf("status = %d (%v)", res.Status, res.Data)
	}

	if premiumOf(t, botID) {
		t.Error("bot still premium after a PayPal dispute")
	}
}

func TestPaypalCaptureLookupFailureRetries(t *testing.T) {
	testdb.Pool(t)
	fakePaypal(t, nil)

	res := HandlePaypalEvent(context.Background(), paypalEvent("CUSTOMER.DISPUTE.CREATED", "dispute",
		`{"disputed_transactions":[{"seller_transaction_id":"CAP-missing"}]}`))

	if res.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 so PayPal retries", res.Status)
	}
}
