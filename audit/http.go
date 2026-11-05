package audit

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
)

var skipOps = map[string]bool{
	"post_bot_stats":    true,
	"post_server_stats": true,
}

var pluralTypes = map[string]string{
	"users":     "user",
	"bots":      "bot",
	"servers":   "server",
	"teams":     "team",
	"packs":     "pack",
	"templates": "server_template",
	"themes":    "theme",
	"tickets":   "ticket",
	"reviews":   "review",
	"emojis":    "pack_emoji",
	"stickers":  "pack_sticker",
	"sounds":    "pack_sound",
}

type cellKey struct{}

type cell struct {
	mu        sync.Mutex
	opID      string
	actorType string
	actorID   string
	data      map[string]any
}

func cellFrom(r *http.Request) *cell {
	c, _ := r.Context().Value(cellKey{}).(*cell)
	return c
}

func Capture(r *http.Request, opID string, auth uapi.AuthData) {
	c := cellFrom(r)

	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.opID = opID

	if auth.Authorized && auth.ID != "" && c.actorID == "" {
		c.actorType = auth.TargetType
		c.actorID = auth.ID
	}
}

func SetActor(r *http.Request, actorType, actorID string) {
	if c := cellFrom(r); c != nil {
		c.mu.Lock()
		c.actorType, c.actorID = actorType, actorID
		c.mu.Unlock()
	}
}

func AddData(r *http.Request, key string, value any) {
	if c := cellFrom(r); c != nil {
		c.mu.Lock()
		if c.data == nil {
			c.data = map[string]any{}
		}
		c.data[key] = value
		c.mu.Unlock()
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}

	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}

	return s.ResponseWriter.Write(b)
}

func target(pattern string, params map[string]string) (string, string) {
	if t, id := params["target_type"], params["target_id"]; t != "" && id != "" {
		return t, id
	}

	segments := strings.Split(strings.Trim(pattern, "/"), "/")

	if len(segments) >= 2 {
		if t, ok := pluralTypes[segments[0]]; ok && strings.HasPrefix(segments[1], "{") {
			return t, params[strings.Trim(segments[1], "{}")]
		}
	}

	return "", ""
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		c := &cell{}
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), cellKey{}, c)))

		c.mu.Lock()
		defer c.mu.Unlock()

		if c.opID == "" || skipOps[c.opID] {
			return
		}

		pattern := ""
		params := map[string]string{}

		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			pattern = rctx.RoutePattern()

			for i, key := range rctx.URLParams.Keys {
				if key != "*" && i < len(rctx.URLParams.Values) {
					params[key] = rctx.URLParams.Values[i]
				}
			}
		}

		targetType, targetID := target(pattern, params)

		data := map[string]any{"method": r.Method, "route": pattern}

		if len(params) > 0 {
			data["params"] = params
		}

		for k, v := range c.data {
			data[k] = v
		}

		status := rec.status

		if status == 0 {
			status = http.StatusOK
		}

		Record(Event{
			Source:     SourceAPI,
			ActorType:  c.actorType,
			ActorID:    c.actorID,
			Action:     c.opID,
			TargetType: targetType,
			TargetID:   targetID,
			Status:     status,
			Data:       data,
		})
	})
}
