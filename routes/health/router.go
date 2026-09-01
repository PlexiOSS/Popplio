package health

import (
	"context"
	"net/http"
	"strings"
	"time"

	arcadiadclient "popplio/arcadia/dclient"
	"popplio/db"
	infernoplexdclient "popplio/infernoplex/dclient"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

// checkTimeout bounds every health check independently of whatever timeout
// (or lack of one) the caller's own request context carries. Every DB-backed
// check below shares state.Pool with the rest of the application -- with no
// bound here, a brief moment of real connection-pool contention leaves
// Pool.Acquire blocking indefinitely, and it's the external monitor's own
// client-side timeout that ends up deciding "down", often well before the
// query would have actually succeeded. That produces exactly the
// down-then-instantly-recovered flapping across multiple, otherwise
// unrelated checks at once (they all queue on the same pool) instead of a
// clean, fast, deterministic 503.
const checkTimeout = 4 * time.Second

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

func dbCheck() func(ctx context.Context) bool {
	return func(ctx context.Context) bool {
		ok, err := db.New(state.Pool).HealthCheck(ctx)

		if err != nil {
			return false
		}

		return ok
	}
}

func tableCheck(table string) func(ctx context.Context) bool {
	return func(ctx context.Context) bool {
		exists, err := db.New(state.Pool).TableExists(ctx, table)

		if err != nil {
			return false
		}

		return exists
	}
}

var services = []service{
	{"/health/api", "API", dbCheck()},
	{"/health/bots", "Bot Listings", tableCheck("bots")},
	{"/health/servers", "Server Listings", tableCheck("servers")},
	{"/health/packs", "Pack Listings", tableCheck("packs")},
	{"/health/blogs", "Blog Service", tableCheck("blogs")},
	{"/health/search", "Search Service", tableCheck("bots")},
	{"/health/auth", "Discord Auth", tableCheck("users")},
	{"/health/tickets", "Support Tickets", tableCheck("tickets")},
	{"/health/staff-panel", "Staff Panel", tableCheck("staff_positions")},

	{"/health/database", "Database", dbCheck()},
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

		uapi.Route{
			Pattern: svc.path,
			OpId:    "health_" + opSuffix + "_head",
			Method:  uapi.HEAD,
			Docs:    docsFor(svc.name),
			Handler: handlerFor(svc.check),
		}.Route(r)
	}
}

func docsFor(name string) func() *docs.Doc {
	return func() *docs.Doc {
		return &docs.Doc{
			Summary:     "Health: " + name,
			Description: "Returns 200 if " + name + " is healthy, 503 if not. No response body to parse status code is the signal.",
		}
	}
}

func handlerFor(check func(ctx context.Context) bool) func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
		ctx, cancel := context.WithTimeout(d.Context, checkTimeout)
		defer cancel()

		if !check(ctx) {
			return uapi.HttpResponse{Status: http.StatusServiceUnavailable}
		}

		return uapi.HttpResponse{Status: http.StatusOK}
	}
}
