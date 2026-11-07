// Copyright (C) 2026 NodeByte LTD

package assets

import "testing"

func TestCreditDiscount(t *testing.T) {
	cases := []struct {
		name                string
		price, offered      int
		wantUsed, wantCents int
	}{
		{"nothing offered", 199, 0, 0, 0},
		{"free item", 0, 100, 0, 0},
		{"negative offer", 199, -10, 0, 0},
		{"small offer", 199, 50, 50, 25},
		{"odd offer rounds down", 199, 51, 50, 25},
		{"capped at half the price", 199, 1000, 198, 99},
		{"cap on an even price", 500, 1000, 500, 250},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			used, cents := CreditDiscount(c.price, c.offered)

			if used != c.wantUsed || cents != c.wantCents {
				t.Fatalf("CreditDiscount(%d, %d) = %d, %d; want %d, %d", c.price, c.offered, used, cents, c.wantUsed, c.wantCents)
			}
		})
	}
}

func TestPriceCents(t *testing.T) {
	if got := PriceCents(1.99); got != 199 {
		t.Fatalf("PriceCents(1.99) = %d, want 199", got)
	}

	if got := PriceCents(4.99); got != 499 {
		t.Fatalf("PriceCents(4.99) = %d, want 499", got)
	}
}

func TestPlanLengths(t *testing.T) {
	want := map[string][2]int{"bronze": {720, 720}, "silver": {4320, 4320}, "gold": {8760, 8784}}

	for _, plan := range Plans {
		w, ok := want[plan.ID]

		if !ok {
			t.Fatalf("unexpected plan %q", plan.ID)
		}

		if plan.TimePeriod != w[0] && plan.TimePeriod != w[1] {
			t.Fatalf("%s lasts %d hours, want %v", plan.ID, plan.TimePeriod, w)
		}
	}
}

func TestLeapYear(t *testing.T) {
	cases := map[int]bool{2024: true, 2026: false, 2028: true, 1900: false, 2000: true, 2100: false, 2400: true}

	for year, want := range cases {
		if got := leapYear(year); got != want {
			t.Errorf("leapYear(%d) = %v, want %v", year, got, want)
		}
	}
}

func TestPremiumLength(t *testing.T) {
	cases := map[int]string{24: "1 day", 720: "30 days", 8784: "366 days", 5: "5 hours", 36: "36 hours"}

	for hours, want := range cases {
		if got := premiumLength(hours); got != want {
			t.Errorf("premiumLength(%d) = %q, want %q", hours, got, want)
		}
	}
}

func TestParseDefaultsToBot(t *testing.T) {
	got := CreatePerkData{ProductName: "bronze", ProductID: "premium", For: "1"}.Parse("9")

	if got.ForType != "bot" || got.UserID != "9" || got.For != "1" {
		t.Fatalf("Parse = %+v", got)
	}

	if s := (CreatePerkData{ForType: "server"}).Parse("9"); s.ForType != "server" {
		t.Fatalf("server for_type became %q", s.ForType)
	}
}
