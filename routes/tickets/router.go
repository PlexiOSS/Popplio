// Package tickets mounts the "Tickets" group of API routes.
//
// These API endpoints are related to tickets on IBL
package tickets

import (
	"popplio/api"
	"popplio/routes/tickets/endpoints/create_ticket"
	"popplio/routes/tickets/endpoints/create_ticket_message"
	"popplio/routes/tickets/endpoints/get_staff_tickets"
	"popplio/routes/tickets/endpoints/get_ticket"
	"popplio/routes/tickets/endpoints/get_ticket_topics"
	"popplio/routes/tickets/endpoints/get_user_tickets"
	"popplio/routes/tickets/endpoints/patch_ticket"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Tickets"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to tickets on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/tickets/{id}",
		OpId:    "get_ticket",
		Method:  uapi.GET,
		Docs:    get_ticket.Docs,
		Handler: get_ticket.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/tickets/topics",
		OpId:    "get_ticket_topics",
		Method:  uapi.GET,
		Docs:    get_ticket_topics.Docs,
		Handler: get_ticket_topics.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/tickets/{id}/messages",
		OpId:    "create_ticket_message",
		Method:  uapi.POST,
		Docs:    create_ticket_message.Docs,
		Handler: create_ticket_message.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/tickets/{id}",
		OpId:    "patch_ticket",
		Method:  uapi.PATCH,
		Docs:    patch_ticket.Docs,
		Handler: patch_ticket.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/tickets",
		OpId:    "get_staff_tickets",
		Method:  uapi.GET,
		Docs:    get_staff_tickets.Docs,
		Handler: get_staff_tickets.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // Staff permission is checked in-handler, same as get_ticket/patch_ticket
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{user_id}/tickets",
		OpId:    "get_user_tickets",
		Method:  uapi.GET,
		Docs:    get_user_tickets.Docs,
		Handler: get_user_tickets.Route,
		Auth: []uapi.AuthType{
			{
				URLVar:       "user_id",
				Type:         api.TargetTypeUser,
				AllowedScope: "ban_exempt",
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{user_id}/tickets",
		OpId:    "create_ticket",
		Method:  uapi.POST,
		Docs:    create_ticket.Docs,
		Handler: create_ticket.Route,
		Auth: []uapi.AuthType{
			{
				URLVar:       "user_id",
				Type:         api.TargetTypeUser,
				AllowedScope: "ban_exempt",
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
