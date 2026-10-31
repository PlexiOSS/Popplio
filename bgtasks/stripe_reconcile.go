// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/db"
	"popplio/promo"
	"popplio/routes/payments/assets"
	"popplio/state"

	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"go.uber.org/zap"
)

const (
	stripeReconcileMaxAge = 30 * 24 * time.Hour
	stripeReconcileSettle = 15 * time.Minute
)

func StripeReconcile(ctx context.Context) error {
	if state.Config.Meta.StripeSecretKey == "" {
		return nil
	}

	q := db.New(state.Pool)

	firstFulfilled, err := q.GetFirstPaymentFulfillmentAt(ctx, "stripe")

	if err != nil {
		return fmt.Errorf("reading first fulfillment: %w", err)
	}

	if !firstFulfilled.Valid {
		return nil
	}

	cutoff := firstFulfilled.Time

	if oldest := time.Now().Add(-stripeReconcileMaxAge); oldest.After(cutoff) {
		cutoff = oldest
	}

	settled := time.Now().Add(-stripeReconcileSettle)

	params := &stripe.CheckoutSessionListParams{}
	params.Limit = stripe.Int64(100)

	iter := session.List(params)

	var checked, reconciled, invalid, failed int

	for iter.Next() {
		s := iter.CheckoutSession()
		created := time.Unix(s.Created, 0)

		if created.Before(cutoff) {
			break
		}

		if created.After(settled) || s.Mode != stripe.CheckoutSessionModePayment {
			continue
		}

		if s.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid && s.PaymentStatus != stripe.CheckoutSessionPaymentStatusNoPaymentRequired {
			continue
		}

		checked++

		exists, err := q.PaymentFulfillmentExists(ctx, db.PaymentFulfillmentExistsParams{Provider: "stripe", ExternalID: s.ID})

		if err != nil {
			return fmt.Errorf("checking fulfillment for %s: %w", s.ID, err)
		}

		if exists {
			continue
		}

		var payload assets.PerkData

		if err := jsonimpl.Unmarshal([]byte(s.ClientReferenceID), &payload); err != nil {
			invalid++
			state.Logger.Error("stripe_reconcile: paid session has an unreadable client reference", zap.String("session", s.ID), zap.Error(err))
			continue
		}

		fulfilled, err := assets.Fulfill(ctx, "stripe", s.ID, payload)

		switch {
		case errors.Is(err, assets.ErrInvalidPurchase):
			invalid++
			state.Logger.Error("stripe_reconcile: paid session can't be applied, needs manual review", zap.String("session", s.ID), zap.Error(err), zap.Any("payload", payload))
		case err != nil:
			failed++
			state.Logger.Error("stripe_reconcile: failed to fulfill paid session", zap.String("session", s.ID), zap.Error(err))
		case fulfilled:
			reconciled++
			promo.Complete(ctx, s.ID)
			state.Logger.Warn("stripe_reconcile: fulfilled a paid session the webhook missed", zap.String("session", s.ID), zap.Any("payload", payload))
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("listing checkout sessions: %w", err)
	}

	state.Logger.Info("stripe_reconcile: done",
		zap.Int("checked", checked),
		zap.Int("reconciled", reconciled),
		zap.Int("invalid", invalid),
		zap.Int("failed", failed),
	)

	if failed > 0 {
		return fmt.Errorf("%d paid sessions could not be fulfilled", failed)
	}

	return nil
}
