// Copyright (C) 2026 NodeByte LTD

package create_session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"popplio/api"

	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
)

func call(limits []string, body string) uapi.HttpResponse {
	req := httptest.NewRequest(http.MethodPost, "/users/1/sessions", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("target_type", "users")
	rctx.URLParams.Add("target_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	return Route(uapi.RouteData{
		Context: req.Context(),
		Auth: uapi.AuthData{
			TargetType: api.TargetTypeUser,
			ID:         "1",
			Authorized: true,
			Data:       map[string]any{"perm_limits": limits},
		},
	}, req)
}

func TestLimitedSessionCannotMintBroaderSession(t *testing.T) {
	limits := []string{"manage_sessions"}

	if res := call(limits, `{"name":"x","type":"api","expiry":60}`); res.Status != http.StatusForbidden {
		t.Errorf("unlimited session from a limited one: status %d, want 403", res.Status)
	}

	if res := call(limits, `{"name":"x","type":"api","expiry":60,"perm_limits":["manage_sessions","edit_bots"]}`); res.Status != http.StatusForbidden {
		t.Errorf("session with a permission the caller lacks: status %d, want 403", res.Status)
	}
}
