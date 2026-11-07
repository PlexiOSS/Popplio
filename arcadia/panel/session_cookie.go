// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"slices"
	"strings"
	"time"

	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	sessionCookieName   = "arcadia_session"
	sessionCookieMaxAge = 30 * 24 * time.Hour
)

var (
	errCookieContentType = errors.New("Requests signed in with the session cookie must be sent as application/json")
	errCookieOrigin      = errors.New("This origin can't use the staff panel session cookie")
)

func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionCookieMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func allowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	return slices.Contains(state.Config.Arcadia.Panel.AllowedOrigins, strings.TrimRight(origin, "/"))
}

func optText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max]
}

func setEmptyToken(obj map[string]json.RawMessage, token string) bool {
	raw, ok := obj["login_token"]

	if !ok {
		return false
	}

	var current string

	if json.Unmarshal(raw, &current) != nil || current != "" {
		return false
	}

	encoded, _ := json.Marshal(token)
	obj["login_token"] = encoded

	return true
}

func injectCookieToken(r *http.Request, body []byte) ([]byte, error) {
	cookie, err := r.Cookie(sessionCookieName)

	if err != nil || cookie.Value == "" {
		return body, nil
	}

	var top map[string]json.RawMessage

	if json.Unmarshal(body, &top) != nil || len(top) != 1 {
		return body, nil
	}

	var name string

	for k := range top {
		name = k
	}

	var inner map[string]json.RawMessage

	if json.Unmarshal(top[name], &inner) != nil {
		return body, nil
	}

	injected := false

	if name == "Authorize" {
		var action map[string]json.RawMessage

		if json.Unmarshal(inner["action"], &action) == nil && len(action) == 1 {
			for variant, payload := range action {
				var fields map[string]json.RawMessage

				if json.Unmarshal(payload, &fields) == nil && setEmptyToken(fields, cookie.Value) {
					action[variant], _ = json.Marshal(fields)
					inner["action"], _ = json.Marshal(action)
					injected = true
				}
			}
		}
	} else {
		injected = setEmptyToken(inner, cookie.Value)
	}

	if !injected {
		return body, nil
	}

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if mediaType != "application/json" {
		return nil, errCookieContentType
	}

	if origin := r.Header.Get("Origin"); origin != "" && !allowedOrigin(origin) {
		return nil, errCookieOrigin
	}

	top[name], _ = json.Marshal(inner)

	return json.Marshal(top)
}
