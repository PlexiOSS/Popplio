// Copyright (C) 2026 NodeByte LTD

package timex

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestMarshalJSONRendersHumanReadableString(t *testing.T) {
	d := Duration(5 * time.Minute)

	b, err := d.MarshalJSON()

	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	if got, want := string(b), `"5m0s"`; got != want {
		t.Errorf("MarshalJSON() = %s, want %s", got, want)
	}
}

func TestUnmarshalJSONFromString(t *testing.T) {
	var d Duration

	if err := d.UnmarshalJSON([]byte(`"5m0s"`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if time.Duration(d) != 5*time.Minute {
		t.Errorf("got %v, want 5m", time.Duration(d))
	}
}

func TestUnmarshalJSONFromNumber(t *testing.T) {
	var d Duration

	// JSON numbers decode to float64; a bare number is taken as a
	// nanosecond count, not a unit-suffixed duration.
	if err := d.UnmarshalJSON([]byte(`300000000000`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if time.Duration(d) != 5*time.Minute {
		t.Errorf("got %v, want 5m", time.Duration(d))
	}
}

func TestUnmarshalJSONRoundTrip(t *testing.T) {
	original := Duration(90 * time.Second)

	b, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var roundTripped Duration
	if err := roundTripped.UnmarshalJSON(b); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if roundTripped != original {
		t.Errorf("round trip = %v, want %v", roundTripped, original)
	}
}

func TestUnmarshalJSONRejectsInvalidDurationString(t *testing.T) {
	var d Duration

	if err := d.UnmarshalJSON([]byte(`"not a duration"`)); err == nil {
		t.Error("expected an error for an unparseable duration string")
	}
}

func TestUnmarshalJSONRejectsOtherTypes(t *testing.T) {
	var d Duration

	cases := []string{`true`, `null`, `["5m"]`, `{"a":1}`}

	for _, c := range cases {
		if err := d.UnmarshalJSON([]byte(c)); err == nil {
			t.Errorf("expected an error for %s", c)
		}
	}
}

func TestUnmarshalJSONRejectsInvalidJSON(t *testing.T) {
	var d Duration

	if err := d.UnmarshalJSON([]byte(`not json at all`)); err == nil {
		t.Error("expected an error for malformed JSON")
	}
}

func TestScanIntervalCombinesAllFields(t *testing.T) {
	var d Duration

	interval := pgtype.Interval{
		Microseconds: int64(3 * time.Second / time.Microsecond),
		Days:         2,
		Months:       1,
		Valid:        true,
	}

	if err := d.ScanInterval(interval); err != nil {
		t.Fatalf("ScanInterval: %v", err)
	}

	want := 3*time.Second + 2*24*time.Hour + 1*30*24*time.Hour

	if time.Duration(d) != want {
		t.Errorf("got %v, want %v", time.Duration(d), want)
	}
}

func TestScanIntervalZero(t *testing.T) {
	var d Duration

	if err := d.ScanInterval(pgtype.Interval{}); err != nil {
		t.Fatalf("ScanInterval: %v", err)
	}

	if time.Duration(d) != 0 {
		t.Errorf("got %v, want 0", time.Duration(d))
	}
}
