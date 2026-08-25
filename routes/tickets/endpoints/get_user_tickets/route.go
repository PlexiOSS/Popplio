// Package get_user_tickets implements GET /users/{user_id}/tickets — "Get
// User Tickets".
//
// Gets every ticket the given user has opened.
package get_user_tickets

import (
	"errors"
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	ticketColsArr = dbutil.GetCols(types.Ticket{})
	ticketCols    = strings.Join(ticketColsArr, ", ")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Tickets",
		Description: "Gets every ticket the given user has opened.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.TicketList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+ticketCols+" FROM tickets WHERE user_id = $1 ORDER BY created_at DESC", d.Auth.ID)

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
		author, err := dovewing.GetUser(d.Context, ticketList[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch ticket author [dovewing]", err, zap.String("userId", d.Auth.ID))
		}

		ticketList[i].Author = author
	}

	return uapi.HttpResponse{
		Json: types.TicketList{Tickets: ticketList},
	}
}
