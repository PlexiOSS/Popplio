// Copyright (C) 2026 NodeByte LTD

package views

import (
	"net/http/httptest"
	"testing"
)

func TestIsCrawler(t *testing.T) {
	for _, ua := range []string{
		"",
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0)",
		"Discordbot/2.0",
		"facebookexternalhit/1.1",
		"python-requests/2.31",
		"curl/8.4.0",
		"Mozilla/5.0 HeadlessChrome/120.0",
	} {
		if !IsCrawler(ua) {
			t.Errorf("%q should count as a crawler", ua)
		}
	}

	for _, ua := range []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
	} {
		if IsCrawler(ua) {
			t.Errorf("%q should count as a visitor", ua)
		}
	}
}

func TestViewerIP(t *testing.T) {
	r := httptest.NewRequest("POST", "/bots/1/views", nil)
	r.RemoteAddr = "203.0.113.7:443"

	if got := ViewerIP(r); got != "203.0.113.7" {
		t.Errorf("with a port = %q", got)
	}

	r.RemoteAddr = "203.0.113.7"
	r.Header.Set("CF-Connecting-IP", "1.1.1.1")

	if got := ViewerIP(r); got != "203.0.113.7" {
		t.Errorf("the Cloudflare header should be left to the clientip middleware, got %q", got)
	}
}

func TestSeenKey(t *testing.T) {
	base := SeenKey("salt", "1.1.1.1", EventView, "bot", "1")

	if base != SeenKey("salt", "1.1.1.1", EventView, "bot", "1") {
		t.Fatal("the same visitor should get the same key on the same day")
	}

	for name, other := range map[string]string{
		"another day":     SeenKey("salt2", "1.1.1.1", EventView, "bot", "1"),
		"another visitor": SeenKey("salt", "2.2.2.2", EventView, "bot", "1"),
		"another event":   SeenKey("salt", "1.1.1.1", EventInvite, "bot", "1"),
		"another type":    SeenKey("salt", "1.1.1.1", EventView, "server", "1"),
		"another listing": SeenKey("salt", "1.1.1.1", EventView, "bot", "2"),
	} {
		if other == base {
			t.Errorf("%s should get a different key", name)
		}
	}
}
