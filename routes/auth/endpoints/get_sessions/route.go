// Package get_sessions implements GET /{target_type}/{target_id}/sessions —
// "Get Sessions".
//
// Gets all session tokens of an entity
package get_sessions

import (
	"net/http"
	"strings"

	"popplio/api/resp"
	"popplio/validators"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Sessions",
		Description: "Gets all session tokens of an entity",
		Resp:        types.SessionList{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The entity type to use",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID to use",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Missing target_id or target_type")
	}

	targetType = strings.TrimSuffix(targetType, "s")

	rows, err := db.New(state.Pool).GetSessions(d.Context, db.GetSessionsParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return resp.Err("Error while getting user tokens", err)
	}

	tokens := make([]*types.Session, len(rows))
	for i, row := range rows {
		tokens[i] = &types.Session{
			ID:         row.ID,
			Name:       row.Name,
			CreatedAt:  row.CreatedAt.Time,
			Type:       row.Type,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			PermLimits: row.PermLimits,
			Expiry:     row.Expiry.Time,
		}
	}

	return uapi.HttpResponse{
		Json: types.SessionList{Sessions: tokens},
	}
}
