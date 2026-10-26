// Copyright (C) 2026 NodeByte LTD

package connect_linked_roles

import (
	"errors"
	"net/http"
	"net/url"

	"popplio/api/resp"
	"popplio/linkedroles"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Connect Linked Roles",
		Description: "Exchanges a Discord OAuth2 code (scopes identify and role_connections.write) and sets the user's Omniplex role connection from their linked-role badges. The redirect URI must be /linked-roles/callback on an allowed frontend.",
		Req:         types.LinkedRolesConnect{},
		Resp:        types.LinkedRolesResult{},
	}
}

func origin(raw string) string {
	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}

	return u.Scheme + "://" + u.Host
}

func allowedRedirect(raw string) bool {
	u, err := url.Parse(raw)

	if err != nil || u.Path != "/linked-roles/callback" || u.RawQuery != "" {
		return false
	}

	o := origin(raw)

	if o == "" {
		return false
	}

	if o == origin(state.Config.Sites.Frontend) {
		return true
	}

	for _, r := range state.Config.DiscordAuth.AllowedRedirects {
		if o == origin(r) {
			return true
		}
	}

	return false
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if !linkedroles.Enabled() {
		return resp.Status(http.StatusServiceUnavailable, "Linked roles aren't set up on this instance")
	}

	var payload types.LinkedRolesConnect

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return resp.BadRequest("code and redirect_uri are required")
	}

	if !allowedRedirect(payload.RedirectURI) {
		return resp.BadRequest("redirect_uri must be /linked-roles/callback on an allowed frontend")
	}

	result, err := linkedroles.Connect(d.Context, payload.Code, payload.RedirectURI)

	if errors.Is(err, linkedroles.ErrDisabled) {
		return resp.Status(http.StatusServiceUnavailable, "Linked roles aren't set up on this instance")
	}

	if err != nil {
		return resp.ErrBody("Failed to connect linked roles", "Discord didn't accept the connection. Try again from the linked roles page.", err)
	}

	return uapi.HttpResponse{Json: types.LinkedRolesResult(*result)}
}
