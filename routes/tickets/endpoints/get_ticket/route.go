package get_ticket

import (
	"encoding/json"
	"errors"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/snowflake/v2"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Ticket",
		Description: "Gets a support ticket. Requires you to be the author of the ticket or have the 'staff' permission",
		Params: []docs.Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "The ticket's ID",
				Required:    true,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Ticket{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	ticketId := chi.URLParam(r, "id")

	if ticketId == "" {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	q := db.New(state.Pool)

	userId, err := q.GetTicketOwner(d.Context, ticketId)

	if err != nil {
		return resp.Err("Error getting ticket", err, zap.String("ticket_id", ticketId))
	}

	if userId != d.Auth.ID {
		sp, err := perms.StaffPerms(d.Context, d.Auth.ID)

		if err != nil {
			return resp.ErrBody("Failed to get user staff perms", "Failed to get user staff perms.", err)
		}

		if !sp.Has(perms.StaffViewTickets) {
			return resp.Forbidden("You do not have permission to view this ticket [" + perms.StaffViewTickets.String() + " is required]")
		}
	}

	row, err := q.GetTicket(d.Context, ticketId)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error getting ticket [collect]", err, zap.String("ticket_id", ticketId))
	}

	var messages []types.Message
	if err := json.Unmarshal(row.Messages, &messages); err != nil {
		return resp.Err("Error parsing ticket messages [json]", err, zap.String("ticket_id", ticketId))
	}

	var ticketContext map[string]string
	if err := json.Unmarshal(row.TicketContext, &ticketContext); err != nil {
		return resp.Err("Error parsing ticket context [json]", err, zap.String("ticket_id", ticketId))
	}

	ticket := types.Ticket{
		ID:            row.ID,
		ChannelID:     row.ChannelID,
		TopicID:       row.TopicID,
		Issue:         row.Issue,
		TicketContext: ticketContext,
		Messages:      messages,
		UserID:        row.UserID,
		CloseUserID:   row.CloseUserID,
		Open:          row.Open,
		CreatedAt:     row.CreatedAt.Time,
		EncKey:        row.EncKey,
	}

	ticket.Author, err = dovewing.GetUser(d.Context, ticket.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error getting ticket author [dovewing]", err, zap.String("ticket_id", ticketId))
	}

	if ticket.CloseUserID.Valid && ticket.CloseUserID.String != "" {
		ticket.CloseUser, err = dovewing.GetUser(d.Context, ticket.CloseUserID.String, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error getting ticket closer [dovewing]", err, zap.String("ticket_id", ticketId))
		}
	}

	for i := range ticket.Messages {
		ticket.Messages[i].Author, err = dovewing.GetUser(d.Context, ticket.Messages[i].AuthorID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error getting ticket message author [dovewing]", err, zap.String("ticket_id", ticketId))
		}

		id, err := snowflake.Parse(ticket.Messages[i].ID)

		if err != nil {
			return resp.Err("Error parsing snowflake", err, zap.String("ticket_id", ticketId))
		}

		ticket.Messages[i].Timestamp = id.Time()
	}

	return uapi.HttpResponse{
		Json: ticket,
	}
}
