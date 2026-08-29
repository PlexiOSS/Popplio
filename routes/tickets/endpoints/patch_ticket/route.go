package patch_ticket

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
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

	q := db.New(state.Pool)

	userID, err := q.GetTicketOwner(d.Context, ticketID)

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

	closeUserID := pgtype.Text{}
	if !payload.Open {
		closeUserID = pgtype.Text{String: d.Auth.ID, Valid: true}
	}

	err = q.UpdateTicketOpenState(d.Context, db.UpdateTicketOpenStateParams{
		Open:        payload.Open,
		CloseUserID: closeUserID,
		ID:          ticketID,
	})

	if err != nil {
		return resp.Err("Failed to update ticket", err, zap.String("ticket_id", ticketID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
