package create_ticket

import (
	"encoding/json"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/tickets"
	"popplio/types"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-playground/validator/v10"

	"github.com/PlexiOSS/Keel/crypto"
	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
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

	messagesJSON, err := json.Marshal([]types.Message{firstMessage})

	if err != nil {
		return resp.Err("Failed to marshal ticket message", err)
	}

	err = db.New(state.Pool).InsertTicket(d.Context, db.InsertTicketParams{
		ID:        ticketID,
		ChannelID: "",
		TopicID:   payload.Topic,
		Issue:     payload.Issue,
		Messages:  messagesJSON,
		UserID:    d.Auth.ID,
	})

	if err != nil {
		return resp.Err("Failed to create ticket", err)
	}

	return uapi.HttpResponse{
		Json: CreatedTicket{ID: ticketID},
	}
}
