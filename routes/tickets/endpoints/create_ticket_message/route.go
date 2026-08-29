package create_ticket_message

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var compiledMessages = uapi.CompileValidationErrors(types.CreateTicketMessage{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Ticket Message",
		Description: "Replies to a ticket. Requires being the ticket's author or staff with the 'manage_tickets' permission. Returns 204 on success.",
		Req:         types.CreateTicketMessage{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The ticket's ID.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	ticketID := chi.URLParam(r, "id")

	if ticketID == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var payload types.CreateTicketMessage

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errs := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errs)
	}

	q := db.New(state.Pool)

	ownerRow, err := q.GetTicketOwnerAndOpen(d.Context, ticketID)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error getting ticket", err, zap.String("ticket_id", ticketID))
	}

	userID, open := ownerRow.UserID, ownerRow.Open

	if userID != d.Auth.ID {
		sp, err := perms.StaffPerms(d.Context, d.Auth.ID)

		if err != nil {
			return resp.ErrBody("Failed to get user staff perms", "Failed to get user staff perms.", err)
		}

		if !sp.Has(perms.StaffManageTickets) {
			return resp.Forbidden("You do not have permission to reply to this ticket [" + perms.StaffManageTickets.String() + " is required]")
		}
	}

	if !open {
		return resp.BadRequest("This ticket is closed. Reopen it before replying.")
	}

	msg := types.Message{
		ID:       snowflake.New(time.Now()).String(),
		Content:  payload.Content,
		AuthorID: d.Auth.ID,
	}

	msgJSON, err := json.Marshal([]types.Message{msg})

	if err != nil {
		return resp.Err("Failed to marshal ticket message", err, zap.String("ticket_id", ticketID))
	}

	err = q.AppendTicketMessage(d.Context, db.AppendTicketMessageParams{
		Messages: msgJSON,
		ID:       ticketID,
	})

	if err != nil {
		return resp.Err("Failed to add ticket message", err, zap.String("ticket_id", ticketID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
