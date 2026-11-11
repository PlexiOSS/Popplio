// Copyright (C) 2026 NodeByte LTD

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"popplio/config"
	"popplio/db"
	"popplio/state"
	"popplio/testdb"

	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestLimitedUserSessionOnAccountRoutes(t *testing.T) {
	testdb.Pool(t)

	prev := state.Config
	state.Config = &config.Config{Sites: config.Sites{Frontend: "https://omniplex.test"}}
	t.Cleanup(func() { state.Config = prev })

	user := testdb.ID(t)
	testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", user, "tok-"+user)
	testdb.Cleanup(t, "DELETE FROM api_sessions WHERE target_id = $1", user)
	testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", user)

	session := func(limits []string) string {
		token := "sess-" + testdb.ID(t) + "-" + testdb.ID(t)
		_, err := db.New(state.Pool).InsertSession(context.Background(), db.InsertSessionParams{
			Token:      token,
			TargetID:   user,
			TargetType: TargetTypeUser,
			Name:       pgtype.Text{String: "test", Valid: true},
			Type:       "api",
			Expiry:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
			PermLimits: limits,
		})
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	route := func(optional bool) uapi.Route {
		return uapi.Route{
			OpId:         "test",
			AuthOptional: optional,
			Auth:         []uapi.AuthType{{URLVar: "id", Type: TargetTypeUser}},
			ExtData:      map[string]any{PERMISSION_CHECK_KEY: nil},
		}
	}

	request := func(token string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/users/"+user+"/data", nil)
		req.Header.Set("Authorization", "User "+token)
		req.Header.Set("Origin", "https://omniplex.test")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", user)
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	limited := session([]string{"view_webhooks"})
	full := session([]string{})

	if _, res, ok := Authorize(route(false), request(limited)); ok || res.Status != http.StatusForbidden {
		t.Errorf("limited session on an account route: ok=%v status=%d, want 403", ok, res.Status)
	}

	if data, _, ok := Authorize(route(false), request(full)); !ok || !data.Authorized {
		t.Errorf("unlimited session on an account route was rejected")
	}

	if data, _, ok := Authorize(route(true), request(limited)); !ok || data.Authorized {
		t.Errorf("limited session on an optional route: ok=%v authorized=%v, want treated as signed out", ok, data.Authorized)
	}
}
