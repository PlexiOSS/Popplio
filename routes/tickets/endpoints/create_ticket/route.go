// Package create_ticket implements POST /users/{user_id}/tickets — "Create
// Ticket".
//
// Opens a new standalone support ticket. Returns the new ticket's ID.
package create_ticket

import (
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/tickets"
	"popplio/types"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-playground/validator/v10"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/uapi"
)

type CreatedTicket struct {
	ID string `json:"id"`
}

var compiledMessages = uapi.CompileValidationErrors(types.CreateTicket{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Ticket",
		Description: "Opens a new standalone support ticket. Returns the new ticket's ID.",
		Req:         types.CreateTicket{},
		Resp:        CreatedTicket{},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user opening the ticket.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      10 * time.Minute,
		MaxRequests: 3,
		Bucket:      "tickets",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err)
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var payload types.CreateTicket

	hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errs := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errs)
	}

	if tickets.FindTopic(payload.Topic) == nil {
		return resp.BadRequest("Invalid topic")
	}

	ticketID := crypto.RandString(64)
	firstMessage := types.Message{
		ID:       snowflake.New(time.Now()).String(),
		Content:  payload.Message,
		AuthorID: d.Auth.ID,
	}

	_, err = state.Pool.Exec(d.Context,
		"INSERT INTO tickets (id, channel_id, topic_id, issue, messages, user_id) VALUES ($1, $2, $3, $4, $5, $6)",
		ticketID, "", payload.Topic, payload.Issue, []types.Message{firstMessage}, d.Auth.ID,
	)

	if err != nil {
		return resp.Err("Failed to create ticket", err)
	}

	return uapi.HttpResponse{
		Json: CreatedTicket{ID: ticketID},
	}
}
