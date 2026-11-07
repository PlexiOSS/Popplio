// Copyright (C) 2026 NodeByte LTD

package patch_staff_ticket

import (
	"net/http"

	"popplio/api/resp"
	staffassets "popplio/routes/staff/assets"
	"popplio/routes/tickets/endpoints/patch_ticket"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	doc := *patch_ticket.Docs()
	doc.Summary = "Staff: Close or Reopen Ticket"
	doc.Description = "Closes or reopens any support ticket. Requires the 'manage_tickets' staff permission. Called through the staff panel."
	return &doc
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = staffassets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	return patch_ticket.Route(d, r)
}
