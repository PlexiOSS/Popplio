package promo

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func intp(v int) *int {
	return &v
}

func TestDiscount(t *testing.T) {
	percent := func(p int32) *Code {
		return &Code{PercentOff: pgtype.Int4{Int32: p, Valid: true}}
	}

	amount := func(a int32) *Code {
		return &Code{AmountOffCents: pgtype.Int4{Int32: a, Valid: true}}
	}

	cases := []struct {
		name     string
		code     *Code
		subtotal int
		want     int
	}{
		{"half of bronze", percent(50), 199, 100},
		{"full percent", percent(100), 299, 299},
		{"fixed under price", amount(100), 299, 100},
		{"fixed over price", amount(1000), 299, 299},
		{"zero subtotal", percent(50), 0, 0},
	}

	for _, c := range cases {
		if got := Discount(c.code, c.subtotal); got != c.want {
			t.Errorf("%s: Discount = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("  summer-25 "); got != "SUMMER-25" {
		t.Fatalf("Normalize = %q", got)
	}
}

func rejected(t *testing.T, err error, want string) {
	t.Helper()

	var rej *Rejection

	if !errors.As(err, &rej) {
		t.Fatalf("expected rejection %q, got %v", want, err)
	}

	if rej.Reason != want {
		t.Fatalf("rejection = %q, want %q", rej.Reason, want)
	}
}

func TestCreateParamsNormalize(t *testing.T) {
	p := CreateParams{Code: "launch", Products: []string{"premium", "premium"}, PercentOff: intp(20), Duration: "forever", DurationMonths: intp(3)}

	if err := p.normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if p.Code != "LAUNCH" || len(p.Products) != 1 || p.Duration != "once" || p.DurationMonths != nil {
		t.Fatalf("premium-only code not normalised: %+v", p)
	}

	p = CreateParams{Code: "FREEMONTH", Products: []string{"autovote"}, PercentOff: intp(100), Duration: "repeating", DurationMonths: intp(2)}

	if err := p.normalize(); err != nil || p.Duration != "repeating" || *p.DurationMonths != 2 {
		t.Fatalf("repeating auto-vote code: %v %+v", err, p)
	}

	past := time.Now().Add(-time.Hour)

	bad := []struct {
		p    CreateParams
		want string
	}{
		{CreateParams{Code: "a!", Products: []string{"premium"}, PercentOff: intp(10)}, "Codes must be 3 to 40 characters of A-Z, 0-9, _ or -"},
		{CreateParams{Code: "OK1", PercentOff: intp(10)}, "Pick at least one product"},
		{CreateParams{Code: "OK1", Products: []string{"shop"}, PercentOff: intp(10)}, `Unknown product "shop"`},
		{CreateParams{Code: "OK1", Products: []string{"premium"}}, "Set either a percentage or a fixed amount off"},
		{CreateParams{Code: "OK1", Products: []string{"premium"}, PercentOff: intp(10), AmountOffCents: intp(10)}, "Set either a percentage or a fixed amount off"},
		{CreateParams{Code: "OK1", Products: []string{"premium"}, PercentOff: intp(101)}, "The percentage must be between 1 and 100"},
		{CreateParams{Code: "OK1", Products: []string{"autovote"}, PercentOff: intp(10), Duration: "repeating"}, "Repeating discounts need between 1 and 36 months"},
		{CreateParams{Code: "OK1", Products: []string{"autovote"}, PercentOff: intp(10), Duration: "weekly"}, "Duration must be once, repeating or forever"},
		{CreateParams{Code: "OK1", Products: []string{"premium"}, PercentOff: intp(10), MaxRedemptions: intp(0)}, "Total uses must be at least 1"},
		{CreateParams{Code: "OK1", Products: []string{"premium"}, PercentOff: intp(10), ExpiresAt: &past}, "The expiry date must be in the future"},
	}

	for _, c := range bad {
		rejected(t, c.p.normalize(), c.want)
	}
}
