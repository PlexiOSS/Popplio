// Package get_entity_badges implements GET
// /{target_type}/{target_id}/badges — "Get Entity Badges".
//
// Gets the badges assigned to an entity. Public and read-only — assigning a
// badge is a staff-only action through Arcadia's panel/RPC layer, not this
// API.
package get_entity_badges

import (
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

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

type entityBadgeRow struct {
	Reason      string    `db:"reason"`
	AwardedBy   string    `db:"awarded_by"`
	CreatedAt   time.Time `db:"created_at"`
	BadgeID     string    `db:"badge_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Icon        string    `db:"icon"`
	Color       string    `db:"color"`
	TargetTypes []string  `db:"target_types"`
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	rows, err := state.Pool.Query(d.Context,
		`SELECT eb.reason, eb.awarded_by, eb.created_at,
                b.id AS badge_id, b.name, b.description, b.icon, b.color, b.target_types
                FROM entity_badges eb
                INNER JOIN badges b ON b.id = eb.badge_id
                WHERE eb.target_type = $1 AND eb.target_id = $2
                ORDER BY eb.created_at ASC`,
		targetType, targetId)

	if err != nil {
		return resp.Err("Failed to fetch entity badges [db fetch]", err)
	}

	badgeRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[entityBadgeRow])

	if err != nil {
		return resp.Err("Failed to fetch entity badges [collect]", err)
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
			CreatedAt: row.CreatedAt,
		})
	}

	return uapi.HttpResponse{
		Json: types.EntityBadgeList{Badges: badges},
	}
}
