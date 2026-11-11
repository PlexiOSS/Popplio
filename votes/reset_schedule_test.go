// Copyright (C) 2026 NodeByte LTD

package votes

import (
	"testing"
	"time"
)

func TestClosingPeriod(t *testing.T) {
	cases := map[string]string{
		"2026-11-11T12:00:00Z": "2026-10-01",
		"2026-12-01T00:05:00Z": "2026-11-01",
		"2027-01-01T00:00:00Z": "2026-12-01",
		"2026-03-31T23:59:00Z": "2026-02-01",
	}

	for now, want := range cases {
		at, _ := time.Parse(time.RFC3339, now)

		if got := ClosingPeriod(at).Format("2006-01-02"); got != want {
			t.Errorf("ClosingPeriod(%s) = %s, want %s", now, got, want)
		}
	}
}

func TestManualResetPeriod(t *testing.T) {
	cases := []struct {
		now     string
		want    string
		counted bool
	}{
		{"2026-11-28T10:00:00Z", "2026-11-01", true},
		{"2026-11-24T00:00:00Z", "2026-11-01", true},
		{"2026-11-23T23:59:00Z", "", false},
		{"2026-11-03T10:00:00Z", "", false},
		{"2026-02-25T12:00:00Z", "2026-02-01", true},
	}

	for _, c := range cases {
		at, _ := time.Parse(time.RFC3339, c.now)
		period, counted := ManualResetPeriod(at)

		if counted != c.counted || (counted && period.Format("2006-01-02") != c.want) {
			t.Errorf("ManualResetPeriod(%s) = %s %v, want %s %v", c.now, period.Format("2006-01-02"), counted, c.want, c.counted)
		}
	}
}
