// Package get_general_vote_credit_tiers implements GET /votes/credit-tiers —
// "Get General Vote Credit Tiers".
//
// Returns a list of all currently available vote credit tiers sorted in
// ascending order
package get_general_vote_credit_tiers

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get General Vote Credit Tiers",
		Description: "Returns a list of all currently available vote credit tiers sorted in ascending order",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type to filter by. If unset, will not filter by target type.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: []types.VoteCreditTier{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := validators.NormalizeTargetType(r.URL.Query().Get("target_type"))

	filter := pgtype.Text{}
	if targetType != "" {
		filter = pgtype.Text{String: targetType, Valid: true}
	}

	rows, err := db.New(state.Pool).GetVoteCreditTiersFiltered(d.Context, filter)

	if err != nil {
		return resp.ErrBody("An error occurred while fetching vote credit tiers", "An error occurred while fetching vote credit tiers.", err)
	}

	vcts := make([]*types.VoteCreditTier, len(rows))
	for i, row := range rows {
		vcts[i] = &types.VoteCreditTier{
			ID:         row.ID,
			TargetType: row.TargetType,
			Position:   int(row.Position),
			Votes:      int(row.Votes),
			Cents:      int(row.Cents),
			CreatedAt:  row.CreatedAt.Time,
		}
	}

	return uapi.HttpResponse{
		Json: vcts,
	}
}
