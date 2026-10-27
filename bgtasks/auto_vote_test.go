package bgtasks

import (
	"testing"
	"time"
)

func TestInSchedule(t *testing.T) {
	monday9 := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		days  []int16
		start int
		end   int
		tz    string
		now   time.Time
		want  bool
	}{
		{"always", nil, 0, 1440, "UTC", monday9, true},
		{"inside window", nil, 8 * 60, 10 * 60, "UTC", monday9, true},
		{"before window", nil, 10 * 60, 12 * 60, "UTC", monday9, false},
		{"end is exclusive", nil, 0, 9 * 60, "UTC", monday9, false},
		{"matching day", []int16{1}, 0, 1440, "UTC", monday9, true},
		{"other day", []int16{0, 6}, 0, 1440, "UTC", monday9, false},
		{"timezone shifts hour", nil, 3 * 60, 4 * 60, "America/Edmonton", monday9, true},
		{"timezone shifts day", []int16{0}, 0, 1440, "Pacific/Pago_Pago", time.Date(2026, 10, 26, 5, 0, 0, 0, time.UTC), true},
		{"bad timezone falls back to UTC", nil, 8 * 60, 10 * 60, "Not/AZone", monday9, true},
	}

	for _, c := range cases {
		if got := inSchedule(c.days, c.start, c.end, c.tz, c.now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
