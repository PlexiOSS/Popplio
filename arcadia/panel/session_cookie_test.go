// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"popplio/state"
)

func cookieRequest(body, contentType, origin string, withCookie bool) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}

	if origin != "" {
		r.Header.Set("Origin", origin)
	}

	if withCookie {
		r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-token"})
	}

	return r
}

func tokenAt(t *testing.T, body []byte, path ...string) string {
	t.Helper()

	var cur any

	if err := json.Unmarshal(body, &cur); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, p := range path {
		cur = cur.(map[string]any)[p]
	}

	s, _ := cur.(string)

	return s
}

func TestInjectCookieToken(t *testing.T) {
	prev := state.Config.Arcadia.Panel.AllowedOrigins
	state.Config.Arcadia.Panel.AllowedOrigins = []string{"https://panel.omniplex.gg"}
	t.Cleanup(func() { state.Config.Arcadia.Panel.AllowedOrigins = prev })

	hello := `{"Hello":{"login_token":"","version":5}}`

	t.Run("fills an empty token from the cookie", func(t *testing.T) {
		out, err := injectCookieToken(cookieRequest(hello, "application/json", "https://panel.omniplex.gg", true), []byte(hello))

		if err != nil {
			t.Fatal(err)
		}

		if got := tokenAt(t, out, "Hello", "login_token"); got != "cookie-token" {
			t.Fatalf("token = %q", got)
		}
	})

	t.Run("fills a nested authorize token", func(t *testing.T) {
		body := `{"Authorize":{"version":5,"action":{"Logout":{"login_token":""}}}}`
		out, err := injectCookieToken(cookieRequest(body, "application/json", "", true), []byte(body))

		if err != nil {
			t.Fatal(err)
		}

		if got := tokenAt(t, out, "Authorize", "action", "Logout", "login_token"); got != "cookie-token" {
			t.Fatalf("token = %q", got)
		}
	})

	t.Run("leaves a token sent in the body alone", func(t *testing.T) {
		body := `{"Hello":{"login_token":"body-token","version":5}}`
		out, err := injectCookieToken(cookieRequest(body, "text/plain", "https://evil.example", true), []byte(body))

		if err != nil {
			t.Fatal(err)
		}

		if got := tokenAt(t, out, "Hello", "login_token"); got != "body-token" {
			t.Fatalf("token = %q", got)
		}
	})

	t.Run("does nothing without a cookie", func(t *testing.T) {
		out, err := injectCookieToken(cookieRequest(hello, "application/json", "", false), []byte(hello))

		if err != nil || string(out) != hello {
			t.Fatalf("out = %s, err = %v", out, err)
		}
	})

	t.Run("refuses a non-JSON request using the cookie", func(t *testing.T) {
		_, err := injectCookieToken(cookieRequest(hello, "text/plain", "https://panel.omniplex.gg", true), []byte(hello))

		if !errors.Is(err, errCookieContentType) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("refuses an origin that isn't allowed", func(t *testing.T) {
		_, err := injectCookieToken(cookieRequest(hello, "application/json", "https://omniplex.gg", true), []byte(hello))

		if !errors.Is(err, errCookieOrigin) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestCORSAllowedOriginGetsCredentials(t *testing.T) {
	prev := state.Config.Arcadia.Panel.AllowedOrigins
	state.Config.Arcadia.Panel.AllowedOrigins = []string{"https://panel.omniplex.gg"}
	t.Cleanup(func() { state.Config.Arcadia.Panel.AllowedOrigins = prev })

	for origin, want := range map[string]string{
		"https://panel.omniplex.gg": "https://panel.omniplex.gg",
		"https://omniplex.gg":       "*",
	} {
		r := httptest.NewRequest(http.MethodOptions, "/", nil)
		r.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		handler(t).ServeHTTP(rec, r)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != want {
			t.Fatalf("%s: allow-origin = %q, want %q", origin, got, want)
		}

		creds := rec.Header().Get("Access-Control-Allow-Credentials") == "true"

		if creds != (want != "*") {
			t.Fatalf("%s: credentials = %v", origin, creds)
		}
	}
}
