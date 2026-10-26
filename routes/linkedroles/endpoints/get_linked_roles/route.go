// Copyright (C) 2026 NodeByte LTD

package get_linked_roles

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/linkedroles"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Linked Roles",
		Description: "Returns whether Discord linked roles are enabled, the application to authorize, and the badges servers can require.",
		Resp:        types.LinkedRolesMeta{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	meta := types.LinkedRolesMeta{
		Enabled:  linkedroles.Enabled(),
		ClientID: state.Config.DiscordAuth.ClientID,
		Badges:   []types.LinkedRoleBadge{},
	}

	rows, err := db.New(state.Pool).ListLinkedRoleBadges(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch linked-role badges [db fetch]", err)
	}

	for i, b := range rows {
		if i >= linkedroles.MaxBadges {
			break
		}

		meta.Badges = append(meta.Badges, types.LinkedRoleBadge{ID: b.ID, Name: b.Name, Description: b.Description})
	}

	return uapi.HttpResponse{Json: meta}
}
