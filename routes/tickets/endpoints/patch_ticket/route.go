// Package patch_ticket implements PATCH /tickets/{id} — "Patch Ticket".
//
// Closes or reopens a ticket. The author may close their own ticket;
// reopening requires staff with the 'manage_tickets' permission. Returns
// 204 on success.
package patch_ticket

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Ticket",
		Description: "Closes or reopens a ticket. The author may close their own ticket; reopening requires staff. Returns 204 on success.",
		Req:         types.PatchTicket{},
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

	var payload types.PatchTicket

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	var userID string

	err := state.Pool.QueryRow(d.Context, "SELECT user_id FROM tickets WHERE id = $1", ticketID).Scan(&userID)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error getting ticket", err, zap.String("ticket_id", ticketID))
	}

	isOwner := userID == d.Auth.ID
	isStaff := false

	if !isOwner {
		sp, err := perms.StaffPerms(d.Context, d.Auth.ID)

		if err != nil {
			return resp.ErrBody("Failed to get user staff perms", "Failed to get user staff perms.", err)
		}

		if !sp.Has(perms.StaffManageTickets) {
			return resp.Forbidden("You do not have permission to manage this ticket [" + perms.StaffManageTickets.String() + " is required]")
		}

		isStaff = true
	}

	if payload.Open && !isStaff {
		return resp.Forbidden("Only staff can reopen a ticket.")
	}

	var closeUserID *string
	if !payload.Open {
		closeUserID = &d.Auth.ID
	}

	_, err = state.Pool.Exec(d.Context, "UPDATE tickets SET open = $1, close_user_id = $2 WHERE id = $3", payload.Open, closeUserID, ticketID)

	if err != nil {
		return resp.Err("Failed to update ticket", err, zap.String("ticket_id", ticketID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
