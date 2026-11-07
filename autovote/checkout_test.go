// Copyright (C) 2026 NodeByte LTD

package autovote

import (
	"testing"

	"github.com/stripe/stripe-go/v75"
)

func TestCheckoutParams(t *testing.T) {
	cases := []struct {
		plan     string
		cents    int64
		interval stripe.PriceRecurringInterval
	}{
		{"monthly", PriceCents, stripe.PriceRecurringIntervalMonth},
		{"", PriceCents, stripe.PriceRecurringIntervalMonth},
		{"yearly", YearlyPriceCents, stripe.PriceRecurringIntervalYear},
	}

	for _, c := range cases {
		t.Run(c.plan, func(t *testing.T) {
			p := checkoutParams("42", c.plan, "https://omniplex.test", nil)
			price := p.LineItems[0].PriceData

			if *price.UnitAmount != c.cents {
				t.Fatalf("amount = %d, want %d", *price.UnitAmount, c.cents)
			}

			if *price.Recurring.Interval != string(c.interval) {
				t.Fatalf("interval = %s, want %s", *price.Recurring.Interval, c.interval)
			}

			if *p.Mode != string(stripe.CheckoutSessionModeSubscription) {
				t.Fatalf("mode = %s", *p.Mode)
			}

			for name, md := range map[string]map[string]string{"session": p.Metadata, "subscription": p.SubscriptionData.Metadata} {
				if md[metadataKey] != metadataVal || md["user_id"] != "42" {
					t.Fatalf("%s metadata = %v", name, md)
				}
			}

			if *p.ClientReferenceID != "autovote:42" {
				t.Fatalf("client reference = %s", *p.ClientReferenceID)
			}

			if *p.SuccessURL != "https://omniplex.test/dashboard?tab=auto-vote&subscribed=1" {
				t.Fatalf("success url = %s", *p.SuccessURL)
			}
		})
	}
}

func TestPlanPriceCents(t *testing.T) {
	if PlanPriceCents("yearly") != 2999 || PlanPriceCents("monthly") != 299 || PlanPriceCents("bogus") != 299 {
		t.Fatal("plan prices changed")
	}

	if YearlyPriceCents >= PriceCents*12 {
		t.Fatal("the yearly plan isn't cheaper than 12 months")
	}
}
