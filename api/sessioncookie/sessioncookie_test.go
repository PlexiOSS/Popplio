package sessioncookie

import (
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	cases := []struct {
		origin, fetchSite string
		want              bool
	}{
		{"", "same-origin", true},
		{"", "same-site", false},
		{"", "cross-site", false},
		{"", "none", false},
		{"", "", false},
		{"https://omniplex.gg", "same-origin", false},
	}

	for _, c := range cases {
		r := httptest.NewRequest("GET", "https://omniplex.gg/api/v1/users/1", nil)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.fetchSite != "" {
			r.Header.Set("Sec-Fetch-Site", c.fetchSite)
		}

		if got := SameOrigin(r); got != c.want {
			t.Errorf("SameOrigin(Origin=%q, Sec-Fetch-Site=%q) = %v, want %v", c.origin, c.fetchSite, got, c.want)
		}
	}
}
