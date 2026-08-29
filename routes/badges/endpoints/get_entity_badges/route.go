// Package get_entity_badges implements GET
// /{target_type}/{target_id}/badges — "Get Entity Badges".
//
// Gets the badges assigned to an entity. Public and read-only — assigning a
// badge is a staff-only action through Arcadia's panel/RPC layer, not this
// API.
package get_entity_badges

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Entity Badges",
		Description: "Gets the badges assigned to a user, bot, server, or team.",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.EntityBadgeList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	badgeRows, err := db.New(state.Pool).GetEntityBadges(d.Context, db.GetEntityBadgesParams{
		TargetType: targetType,
		TargetID:   targetId,
	})

	if err != nil {
		return resp.Err("Failed to fetch entity badges [db fetch]", err)
	}

	badges := make([]types.EntityBadge, 0, len(badgeRows))

	for _, row := range badgeRows {
		badges = append(badges, types.EntityBadge{
			Badge: types.BadgeCatalog{
				ID:          row.BadgeID,
				Name:        row.Name,
				Description: row.Description,
				Icon:        row.Icon,
				Color:       row.Color,
				TargetTypes: row.TargetTypes,
			},
			Reason:    row.Reason,
			AwardedBy: row.AwardedBy,
			CreatedAt: row.CreatedAt.Time,
		})
	}

	return uapi.HttpResponse{
		Json: types.EntityBadgeList{Badges: badges},
	}
}
