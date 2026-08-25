// Package get_general_vote_credit_tiers implements GET /votes/credit-tiers —
// "Get General Vote Credit Tiers".
//
// Returns a list of all currently available vote credit tiers sorted in
// ascending order
package get_general_vote_credit_tiers

import (
	"errors"
	"net/http"
	"strings"

	"popplio/api/resp"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	voteCreditTiersColsArr = dbutil.GetCols(types.VoteCreditTier{})
	voteCreditTiersCols    = strings.Join(voteCreditTiersColsArr, ",")
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

	var rows pgx.Rows
	var err error
	if targetType == "" {
		rows, err = state.Pool.Query(d.Context, "SELECT "+voteCreditTiersCols+" FROM vote_credit_tiers ORDER BY position ASC")
	} else {
		rows, err = state.Pool.Query(d.Context, "SELECT "+voteCreditTiersCols+" FROM vote_credit_tiers WHERE target_type = $1 ORDER BY position ASC", targetType)
	}

	if err != nil {
		return resp.ErrBody("An error occurred while fetching vote credit tiers", "An error occurred while fetching vote credit tiers.", err)
	}

	vcts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.VoteCreditTier])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{
			Json: []types.VoteCreditTier{},
		}
	}

	return uapi.HttpResponse{
		Json: vcts,
	}
}
