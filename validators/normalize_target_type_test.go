package validators

import "testing"

func TestNormalizeTargetType(t *testing.T) {
	cases := map[string]string{
		"bots":    "bot",
		"bot":     "bot",
		"users":   "user",
		"user":    "user",
		"servers": "server",
		"server":  "server",
		"teams":   "team",
		"team":    "team",
		"packs":   "pack",
		"pack":    "pack",
		// Unknown types fall through to a generic trailing-"s" strip.
		"widgets": "widget",
		"widget":  "widget",
		"":        "",
	}

	for in, want := range cases {
		if got := NormalizeTargetType(in); got != want {
			t.Errorf("NormalizeTargetType(%q) = %q, want %q", in, got, want)
		}
	}
}
