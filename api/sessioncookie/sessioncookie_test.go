package sessioncookie

import "testing"

func TestDomain(t *testing.T) {
	cases := []struct {
		frontend, api, want string
	}{
		{"https://omniplex.gg", "https://api.omniplex.gg", "omniplex.gg"},
		{"https://omniplex.gg", "https://spider.omniplex.gg", "omniplex.gg"},
		{"https://www.omniplex.gg", "https://api.omniplex.gg", "omniplex.gg"},
		{"https://omniplex.gg", "https://omniplex.gg", "omniplex.gg"},
		{"http://localhost:3000", "http://localhost:8080", ""},
		{"http://127.0.0.1:3000", "http://127.0.0.1:8080", ""},
		{"https://omniplex.gg", "https://api.example.com", ""},
		{"https://a.example.com", "https://b.example.com", "example.com"},
	}

	for _, c := range cases {
		if got := Domain(c.frontend, c.api); got != c.want {
			t.Errorf("Domain(%q, %q) = %q, want %q", c.frontend, c.api, got, c.want)
		}
	}
}
