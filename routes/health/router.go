package health

import (
	"context"
	"net/http"
	"strings"

	arcadiadclient "popplio/arcadia/dclient"
	infernoplexdclient "popplio/infernoplex/dclient"
	"popplio/state"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
)

const tagName = "Health"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Lightweight per-subsystem health checks for external uptime monitoring. Each returns 200 when healthy and 503 when not — no response body needed."
}

type service struct {
	path  string
	name  string
	check func(ctx context.Context) bool
}

func dbCheck(query string) func(ctx context.Context) bool {
	return func(ctx context.Context) bool {
		var ok bool

		if err := state.Pool.QueryRow(ctx, query).Scan(&ok); err != nil {
			return false
		}

		return ok
	}
}

var services = []service{
	{"/health/api", "API", dbCheck("SELECT true")},
	{"/health/bots", "Bot Listings", dbCheck("SELECT EXISTS(SELECT 1 FROM bots LIMIT 1)")},
	{"/health/servers", "Server Listings", dbCheck("SELECT EXISTS(SELECT 1 FROM servers LIMIT 1)")},
	{"/health/packs", "Pack Listings", dbCheck("SELECT EXISTS(SELECT 1 FROM packs LIMIT 1)")},
	{"/health/blogs", "Blog Service", dbCheck("SELECT EXISTS(SELECT 1 FROM blogs LIMIT 1)")},
	{"/health/search", "Search Service", dbCheck("SELECT EXISTS(SELECT 1 FROM bots LIMIT 1)")},
	{"/health/auth", "Discord Auth", dbCheck("SELECT EXISTS(SELECT 1 FROM users LIMIT 1)")},
	{"/health/tickets", "Support Tickets", dbCheck("SELECT EXISTS(SELECT 1 FROM tickets LIMIT 1)")},
	{"/health/staff-panel", "Staff Panel", dbCheck("SELECT EXISTS(SELECT 1 FROM staff_positions LIMIT 1)")},

	{"/health/database", "Database", dbCheck("SELECT true")},
	{"/health/infernoplex", "Infernoplex", func(_ context.Context) bool { return infernoplexdclient.Ready() }},
	{"/health/arcadia", "Arcadia", func(_ context.Context) bool { return arcadiadclient.Ready() }},
}

func (b Router) Routes(r *chi.Mux) {
	for _, svc := range services {
		opSuffix := strings.ReplaceAll(svc.path[len("/health/"):], "-", "_")

		uapi.Route{
			Pattern: svc.path,
			OpId:    "health_" + opSuffix,
			Method:  uapi.GET,
			Docs:    docsFor(svc.name),
			Handler: handlerFor(svc.check),
		}.Route(r)
	}
}

func docsFor(name string) func() *docs.Doc {
	return func() *docs.Doc {
		return &docs.Doc{
			Summary:     "Health: " + name,
			Description: "Returns 200 if " + name + " is healthy, 503 if not. No response body to parse — status code is the signal.",
			Resp:        []byte(""),
		}
	}
}

func handlerFor(check func(ctx context.Context) bool) func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
		if !check(d.Context) {
			return uapi.HttpResponse{Status: http.StatusServiceUnavailable}
		}

		return uapi.HttpResponse{Status: http.StatusOK}
	}
}
