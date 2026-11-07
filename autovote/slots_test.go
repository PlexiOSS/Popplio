// Copyright (C) 2026 NodeByte LTD

package autovote

import (
	"context"
	"fmt"
	"testing"

	"popplio/db"
	"popplio/testdb"

	"github.com/stripe/stripe-go/v75"
)

func base(interval stripe.PriceRecurringInterval) *stripe.SubscriptionItem {
	return &stripe.SubscriptionItem{Quantity: 1, Price: &stripe.Price{Recurring: &stripe.PriceRecurring{Interval: interval}}}
}

func slots(n int64) *stripe.SubscriptionItem {
	return &stripe.SubscriptionItem{Quantity: n, Metadata: map[string]string{metadataKey: slotMetadataVal}, Price: &stripe.Price{Recurring: &stripe.PriceRecurring{Interval: stripe.PriceRecurringIntervalMonth}}}
}

func TestSlotsOf(t *testing.T) {
	cases := []struct {
		name         string
		sub          *stripe.Subscription
		wantInterval string
		wantExtra    int
	}{
		{"nil", nil, "month", 0},
		{"no items", &stripe.Subscription{}, "month", 0},
		{"monthly", &stripe.Subscription{Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{base(stripe.PriceRecurringIntervalMonth)}}}, "month", 0},
		{"yearly with slots", &stripe.Subscription{Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{base(stripe.PriceRecurringIntervalYear), slots(4)}}}, "year", 4},
		{"slots item doesn't set the interval", &stripe.Subscription{Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{slots(2), base(stripe.PriceRecurringIntervalYear)}}}, "year", 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			interval, extra := slotsOf(c.sub)

			if interval != c.wantInterval || extra != c.wantExtra {
				t.Fatalf("slotsOf = %s, %d; want %s, %d", interval, extra, c.wantInterval, c.wantExtra)
			}
		})
	}
}

func TestSlotPricing(t *testing.T) {
	if Limit(0) != 3 || Limit(MaxExtraSlots) != 10 {
		t.Fatalf("limits = %d, %d; want 3, 10", Limit(0), Limit(MaxExtraSlots))
	}

	if SlotPrice("month") != 99 || SlotPrice("year") != 999 || SlotPrice("") != 99 {
		t.Fatal("slot prices changed")
	}

	if YearlySlotCents >= SlotPriceCents*12 {
		t.Fatal("a yearly slot isn't cheaper than 12 monthly ones")
	}
}

func TestDueTargetsRespectSlots(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(testdb.Pool(t))
	user := seedSubscriber(t)

	for i := 1; i <= 5; i++ {
		testdb.Exec(t, `INSERT INTO auto_vote_targets (user_id, target_type, target_id, created_at) VALUES ($1, 'bot', $2, NOW() - make_interval(mins => $3))`, user, fmt.Sprint(i), 10-i)
	}

	due := func() []string {
		rows, err := q.ListDueAutoVoteTargets(ctx)

		if err != nil {
			t.Fatal(err)
		}

		var ids []string

		for _, row := range rows {
			if row.UserID == user {
				ids = append(ids, row.TargetID)
			}
		}

		return ids
	}

	if got := due(); fmt.Sprint(got) != "[1 2 3]" {
		t.Fatalf("due with no extra slots = %v, want the 3 oldest", got)
	}

	if extra, err := q.GetAutoVoteExtraSlots(ctx, user); err != nil || extra != 0 {
		t.Fatalf("extra slots = %d, %v", extra, err)
	}

	if err := q.SetAutoVoteExtraSlots(ctx, db.SetAutoVoteExtraSlotsParams{UserID: user, ExtraSlots: 2}); err != nil {
		t.Fatal(err)
	}

	if got := due(); len(got) != 5 {
		t.Fatalf("due with 2 extra slots = %v, want all 5", got)
	}

	if extra, err := q.GetAutoVoteExtraSlots(ctx, testdb.ID(t)); err != nil || extra != 0 {
		t.Fatalf("extra slots for a non-subscriber = %d, %v; want 0", extra, err)
	}
}

func TestBoosterGetsAnExtraDueTarget(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(testdb.Pool(t))
	user := seedSubscriber(t)
	testdb.Cleanup(t, "DELETE FROM booster_perks WHERE user_id = $1", user)

	for i := 1; i <= 5; i++ {
		testdb.Exec(t, `INSERT INTO auto_vote_targets (user_id, target_type, target_id, created_at) VALUES ($1, 'bot', $2, NOW() - make_interval(mins => $3))`, user, fmt.Sprint(i), 10-i)
	}

	count := func() int {
		rows, err := q.ListDueAutoVoteTargets(ctx)

		if err != nil {
			t.Fatal(err)
		}

		n := 0

		for _, row := range rows {
			if row.UserID == user {
				n++
			}
		}

		return n
	}

	if BoosterBonus(ctx, user) != 0 || count() != 3 {
		t.Fatalf("non-booster: bonus %d, due %d; want 0, 3", BoosterBonus(ctx, user), count())
	}

	testdb.Exec(t, "INSERT INTO booster_perks (user_id, boosting) VALUES ($1, true)", user)

	if BoosterBonus(ctx, user) != 1 || count() != 4 {
		t.Fatalf("booster: bonus %d, due %d; want 1, 4", BoosterBonus(ctx, user), count())
	}
}
