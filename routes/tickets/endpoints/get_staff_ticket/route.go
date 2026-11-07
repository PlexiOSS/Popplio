// Copyright (C) 2026 NodeByte LTD

package get_staff_ticket

import (
	"net/http"

	"popplio/api/resp"
	staffassets "popplio/routes/staff/assets"
	"popplio/routes/tickets/endpoints/get_ticket"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	doc := *get_ticket.Docs()
	doc.Summary = "Staff: Get Ticket"
	doc.Description = "Gets any support ticket. Requires the 'view_tickets' staff permission. Called through the staff panel."
	return &doc
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = staffassets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	return get_ticket.Route(d, r)
}
