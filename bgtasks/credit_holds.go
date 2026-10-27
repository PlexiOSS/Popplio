// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"
	"time"

	"popplio/db"
	"popplio/routes/payments/assets"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"go.uber.org/zap"
)

const (
	stripeHoldMinAge = time.Hour
	paypalHoldMinAge = 9 * time.Hour
)

func CreditHoldSweep(ctx context.Context) error {
	q := db.New(state.Pool)

	holds, err := q.ListStalePaymentCreditHolds(ctx, pgtype.Timestamptz{Time: time.Now().Add(-stripeHoldMinAge), Valid: true})

	if err != nil {
		return fmt.Errorf("listing credit holds: %w", err)
	}

	var failed int

	for _, hold := range holds {
		switch hold.Provider {
		case "stripe":
			if state.Config.Meta.StripeSecretKey == "" {
				continue
			}

			s, err := session.Get(hold.ExternalID, nil)

			if err != nil {
				state.Logger.Warn("Credit hold sweep: failed to fetch Stripe session", zap.Error(err), zap.String("session", hold.ExternalID))
				continue
			}

			if s.Status != stripe.CheckoutSessionStatusExpired {
				continue
			}
		case "paypal":
			if time.Since(hold.CreatedAt.Time) < paypalHoldMinAge {
				continue
			}
		default:
			continue
		}

		if err := assets.RefundCreditHold(ctx, hold); err != nil {
			state.Logger.Error("Credit hold sweep: failed to refund credits", zap.Error(err), zap.String("provider", hold.Provider), zap.String("external_id", hold.ExternalID))
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("failed to refund %d credit holds", failed)
	}

	return nil
}
