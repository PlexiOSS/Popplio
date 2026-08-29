// Package get_staff_tickets implements GET /staff/tickets — "Get Staff
// Tickets".
//
// Gets every ticket platform-wide, for staff who can already view any
// ticket via GET /tickets/{id} but had no way to find one to look at.
package get_staff_tickets

import (
	"encoding/json"
	"net/http"
	"strconv"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 20

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Staff Tickets",
		Description: "Gets every ticket platform-wide. Requires the 'view_tickets' staff permission.",
		Params: []docs.Parameter{
			{
				Name:        "open",
				Description: "Filter to open (true) or closed (false) tickets only. Omit for all tickets.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "page",
				Description: "The page number, 1-indexed.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.TicketList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	sp, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.ErrBody("Failed to get user staff perms", "Failed to get user staff perms.", err)
	}

	if !sp.Has(perms.StaffViewTickets) {
		return resp.Forbidden("You do not have permission to view tickets [" + perms.StaffViewTickets.String() + " is required]")
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err != nil || page < 1 {
		page = 1
	}

	openFilter := pgtype.Bool{}
	switch r.URL.Query().Get("open") {
	case "true":
		openFilter = pgtype.Bool{Bool: true, Valid: true}
	case "false":
		openFilter = pgtype.Bool{Bool: false, Valid: true}
	}

	rows, err := db.New(state.Pool).GetStaffTicketsPage(d.Context, db.GetStaffTicketsPageParams{
		Limit:  perPage,
		Offset: int32((page - 1) * perPage),
		Open:   openFilter,
	})

	if err != nil {
		return resp.Err("Failed to fetch tickets [db fetch]", err, zap.String("userId", d.Auth.ID))
	}

	ticketList := make([]types.Ticket, len(rows))
	for i, row := range rows {
		var messages []types.Message
		if err := json.Unmarshal(row.Messages, &messages); err != nil {
			return resp.Err("Failed to parse ticket messages [json]", err, zap.String("userId", d.Auth.ID))
		}

		var ticketContext map[string]string
		if err := json.Unmarshal(row.TicketContext, &ticketContext); err != nil {
			return resp.Err("Failed to parse ticket context [json]", err, zap.String("userId", d.Auth.ID))
		}

		ticketList[i] = types.Ticket{
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
	}

	for i := range ticketList {
		// Best-effort: this spans every ticket ever filed, including ones
		// tied to accounts that no longer exist on Discord (deleted
		// accounts) — that's an expected case here, not a reason to fail
		// the whole list. Leave Author nil rather than erroring out.
		author, err := dovewing.GetUser(d.Context, ticketList[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Warn("Failed to resolve ticket author", zap.Error(err), zap.String("ticketId", ticketList[i].ID), zap.String("authorId", ticketList[i].UserID))
			continue
		}

		ticketList[i].Author = author
	}

	return uapi.HttpResponse{
		Json: types.TicketList{Tickets: ticketList},
	}
}
