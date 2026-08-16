// Package get_staff_tickets implements GET /staff/tickets — "Get Staff
// Tickets".
//
// Gets every ticket platform-wide, for staff who can already view any
// ticket via GET /tickets/{id} but had no way to find one to look at.
package get_staff_tickets

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const perPage = 20

var (
	ticketColsArr = db.GetCols(types.Ticket{})
	ticketCols    = strings.Join(ticketColsArr, ", ")
)

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

	query := "SELECT " + ticketCols + " FROM tickets"
	args := []any{}

	switch r.URL.Query().Get("open") {
	case "true":
		query += " WHERE open = true"
	case "false":
		query += " WHERE open = false"
	}

	query += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
	args = append(args, perPage, (page-1)*perPage)

	rows, err := state.Pool.Query(d.Context, query, args...)

	if err != nil {
		return resp.Err("Failed to fetch tickets [db fetch]", err, zap.String("userId", d.Auth.ID))
	}

	ticketList, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Ticket])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Json: types.TicketList{Tickets: []types.Ticket{}},
		}
	}

	if err != nil {
		return resp.Err("Failed to fetch tickets [collect]", err, zap.String("userId", d.Auth.ID))
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
