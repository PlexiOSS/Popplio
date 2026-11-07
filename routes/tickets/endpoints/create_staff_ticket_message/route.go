// Copyright (C) 2026 NodeByte LTD

package create_staff_ticket_message

import (
	"net/http"

	"popplio/api/resp"
	staffassets "popplio/routes/staff/assets"
	"popplio/routes/tickets/endpoints/create_ticket_message"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	doc := *create_ticket_message.Docs()
	doc.Summary = "Staff: Reply to Ticket"
	doc.Description = "Replies to any support ticket as the signed-in staff member. Requires the 'manage_tickets' staff permission. Called through the staff panel."
	return &doc
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = staffassets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	return create_ticket_message.Route(d, r)
}
