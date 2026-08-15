// Package get_ticket_topics implements GET /tickets/topics — "Get Ticket
// Topics".
//
// Returns the predefined list of topics a ticket can be filed under.
package get_ticket_topics

import (
	"net/http"
	"popplio/tickets"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Ticket Topics",
		Description: "Returns the predefined list of topics a ticket can be filed under.",
		Resp:        types.TicketTopicList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: types.TicketTopicList{
			Topics: tickets.Topics,
		},
	}
}
