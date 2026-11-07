// Copyright (C) 2026 NodeByte LTD

package autovote

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"popplio/db"
	"popplio/state"

	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/product"
	"github.com/stripe/stripe-go/v75/subscription"
	"github.com/stripe/stripe-go/v75/subscriptionitem"
)

var (
	ErrSlotsInactive = errors.New("auto-vote subscription isn't active")
	ErrSlotsInUse    = errors.New("more listings are set up than the new slot count allows")
)

const BoosterSlots = 1

func Limit(extraSlots int) int {
	return MaxTargets + extraSlots
}

func BoosterBonus(ctx context.Context, userID string) int {
	boosting, err := db.New(state.Pool).IsBoosting(ctx, userID)

	if err != nil || !boosting {
		return 0
	}

	return BoosterSlots
}

func SlotPrice(interval string) int {
	if interval == "year" {
		return YearlySlotCents
	}

	return SlotPriceCents
}

func slotsOf(sub *stripe.Subscription) (interval string, extra int) {
	interval = "month"

	if sub == nil || sub.Items == nil {
		return interval, 0
	}

	for _, item := range sub.Items.Data {
		if item.Metadata[metadataKey] == slotMetadataVal {
			extra += int(item.Quantity)
			continue
		}

		if item.Price != nil && item.Price.Recurring != nil && item.Price.Recurring.Interval == stripe.PriceRecurringIntervalYear {
			interval = "year"
		}
	}

	return interval, extra
}

func slotItem(sub *stripe.Subscription) *stripe.SubscriptionItem {
	if sub.Items == nil {
		return nil
	}

	for _, item := range sub.Items.Data {
		if item.Metadata[metadataKey] == slotMetadataVal {
			return item
		}
	}

	return nil
}

var (
	slotProductMu sync.Mutex
	slotProductID string
)

func slotProduct() (string, error) {
	slotProductMu.Lock()
	defer slotProductMu.Unlock()

	if slotProductID != "" {
		return slotProductID, nil
	}

	iter := product.Search(&stripe.ProductSearchParams{
		SearchParams: stripe.SearchParams{Query: fmt.Sprintf("metadata['%s']:'%s' AND active:'true'", metadataKey, slotMetadataVal)},
	})

	for iter.Next() {
		slotProductID = iter.Product().ID
		return slotProductID, nil
	}

	if err := iter.Err(); err != nil {
		return "", err
	}

	p, err := product.New(&stripe.ProductParams{
		Name:        stripe.String("Omniplex Auto-vote extra slot"),
		Description: stripe.String("One more bot or server on your Auto-vote subscription."),
		Metadata:    map[string]string{metadataKey: slotMetadataVal},
	})

	if err != nil {
		return "", err
	}

	slotProductID = p.ID

	return slotProductID, nil
}

func SetExtraSlots(ctx context.Context, userID string, want int) (int, error) {
	q := db.New(state.Pool)

	row, err := q.GetAutoVoteSubscription(ctx, userID)

	if err != nil {
		return 0, ErrSlotsInactive
	}

	if !Active(row.Status, row.CurrentPeriodEnd) {
		return 0, ErrSlotsInactive
	}

	targets, err := q.ListAutoVoteTargets(ctx, userID)

	if err != nil {
		return 0, err
	}

	if len(targets) > Limit(want)+BoosterBonus(ctx, userID) {
		return len(targets), ErrSlotsInUse
	}

	sub, err := subscription.Get(row.StripeSubscriptionID, nil)

	if err != nil {
		return 0, err
	}

	interval, current := slotsOf(sub)
	item := slotItem(sub)

	if want == current {
		return 0, q.SetAutoVoteExtraSlots(ctx, db.SetAutoVoteExtraSlotsParams{UserID: userID, ExtraSlots: int32(want)})
	}

	charge := stripe.String("always_invoice")
	credit := stripe.String("create_prorations")
	strict := stripe.String("error_if_incomplete")

	switch {
	case item == nil:
		productID, err := slotProduct()

		if err != nil {
			return 0, err
		}

		_, err = subscriptionitem.New(&stripe.SubscriptionItemParams{
			Subscription: stripe.String(sub.ID),
			PriceData: &stripe.SubscriptionItemPriceDataParams{
				Currency:   stripe.String(string(stripe.CurrencyUSD)),
				Product:    stripe.String(productID),
				UnitAmount: stripe.Int64(int64(SlotPrice(interval))),
				Recurring: &stripe.SubscriptionItemPriceDataRecurringParams{
					Interval: stripe.String(interval),
				},
			},
			Quantity:          stripe.Int64(int64(want)),
			Metadata:          map[string]string{metadataKey: slotMetadataVal, "user_id": userID},
			ProrationBehavior: charge,
			PaymentBehavior:   strict,
		})

		if err != nil {
			return 0, err
		}
	case want == 0:
		if _, err := subscriptionitem.Del(item.ID, &stripe.SubscriptionItemParams{ProrationBehavior: credit}); err != nil {
			return 0, err
		}
	case want > current:
		if _, err := subscriptionitem.Update(item.ID, &stripe.SubscriptionItemParams{
			Quantity:          stripe.Int64(int64(want)),
			ProrationBehavior: charge,
			PaymentBehavior:   strict,
		}); err != nil {
			return 0, err
		}
	default:
		if _, err := subscriptionitem.Update(item.ID, &stripe.SubscriptionItemParams{
			Quantity:          stripe.Int64(int64(want)),
			ProrationBehavior: credit,
		}); err != nil {
			return 0, err
		}
	}

	return 0, q.SetAutoVoteExtraSlots(ctx, db.SetAutoVoteExtraSlotsParams{UserID: userID, ExtraSlots: int32(want)})
}
