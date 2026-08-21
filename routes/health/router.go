package health

import (
	"context"
	"fmt"
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

func tableCheck(table string) func(ctx context.Context) bool {
	return dbCheck(fmt.Sprintf(
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '%s')",
		table,
	))
}

var services = []service{
	{"/health/api", "API", dbCheck("SELECT true")},
	{"/health/bots", "Bot Listings", tableCheck("bots")},
	{"/health/servers", "Server Listings", tableCheck("servers")},
	{"/health/packs", "Pack Listings", tableCheck("packs")},
	{"/health/blogs", "Blog Service", tableCheck("blogs")},
	{"/health/search", "Search Service", tableCheck("bots")},
	{"/health/auth", "Discord Auth", tableCheck("users")},
	{"/health/tickets", "Support Tickets", tableCheck("tickets")},
	{"/health/staff-panel", "Staff Panel", tableCheck("staff_positions")},

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
		if !check(d.Context) {
			return uapi.HttpResponse{Status: http.StatusServiceUnavailable}
		}

		return uapi.HttpResponse{Status: http.StatusOK}
	}
}
