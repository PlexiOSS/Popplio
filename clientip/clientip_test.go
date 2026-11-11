// Copyright (C) 2026 NodeByte LTD

package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func request(remote string, headers map[string]string) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = remote
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name    string
		remote  string
		headers map[string]string
		want    string
	}{
		{"behind Traefik and Cloudflare", "172.18.0.5:41000", map[string]string{"X-Real-IP": "104.22.31.62", "CF-Connecting-IP": "203.0.113.7"}, "203.0.113.7"},
		{"Cloudflare connecting directly", "104.22.31.62:443", map[string]string{"CF-Connecting-IP": "203.0.113.7"}, "203.0.113.7"},
		{"Cloudflare over IPv6", "[2606:4700::1]:443", map[string]string{"CF-Connecting-IP": "2001:db8::9"}, "2001:db8::9"},
		{"spoofed Cloudflare header, not from Cloudflare", "198.51.100.4:5000", map[string]string{"CF-Connecting-IP": "1.1.1.1"}, "198.51.100.4"},
		{"spoofed header through Traefik, not from Cloudflare", "172.18.0.5:41000", map[string]string{"X-Real-IP": "198.51.100.4", "CF-Connecting-IP": "1.1.1.1"}, "198.51.100.4"},
		{"X-Real-IP ignored from a public peer", "198.51.100.4:5000", map[string]string{"X-Real-IP": "104.22.31.62", "CF-Connecting-IP": "1.1.1.1"}, "198.51.100.4"},
		{"forged internal header without the key", "198.51.100.4:5000", map[string]string{"X-Omniplex-Client-IP": "9.9.9.9", "X-Omniplex-Client-Key": "guess"}, "198.51.100.4"},
		{"X-Forwarded-For is never trusted", "172.18.0.5:41000", map[string]string{"X-Forwarded-For": "9.9.9.9"}, "172.18.0.5"},
		{"no port", "198.51.100.4", nil, "198.51.100.4"},
	}

	for _, c := range cases {
		if got := Resolve(request(c.remote, c.headers)); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestInternalProxy(t *testing.T) {
	out := httptest.NewRequest("GET", "/", nil)
	SetInternal(out, "203.0.113.7")
	out.RemoteAddr = "127.0.0.1:5000"

	if got := Resolve(out); got != "203.0.113.7" {
		t.Fatalf("in-process proxy = %s", got)
	}

	var seen *http.Request
	Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r })).ServeHTTP(httptest.NewRecorder(), out)

	if seen.RemoteAddr != "203.0.113.7" || seen.Header.Get("X-Omniplex-Client-Key") != "" {
		t.Fatalf("middleware left RemoteAddr=%s key=%q", seen.RemoteAddr, seen.Header.Get("X-Omniplex-Client-Key"))
	}
}
