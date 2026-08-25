// Package get_partners implements GET /list/partners — "Get List Partners".
//
// Gets the official partners of the list
package get_partners

import (
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	// Partners
	partnersColsArr = dbutil.GetCols(types.Partner{})
	partnersCols    = strings.Join(partnersColsArr, ",")

	// Partner types
	partnerTypesColsArr = dbutil.GetCols(types.PartnerTypes{})
	partnerTypesCols    = strings.Join(partnerTypesColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Partners",
		Description: "Gets the official partners of the list",
		Resp:        types.PartnerList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+partnersCols+" FROM partners ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch partner list [db fetch]", err)
	}

	defer rows.Close()

	partners, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Partner])

	if err != nil {
		return resp.Err("Failed to fetch partner list [db fetch]", err)
	}

	for i := range partners {
		err := state.Validator.Struct(partners[i])

		if err != nil {
			return resp.ErrBody("Failed to validate partner", "Could not validate partner "+partners[i].ID+".", err, zap.String("partner_id", partners[i].ID))
		}

		partners[i].User, err = dovewing.GetUser(d.Context, partners[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch partner user", err, zap.String("partner_id", partners[i].ID), zap.String("user_id", partners[i].UserID))
		}

	}

	rows, err = state.Pool.Query(d.Context, "SELECT "+partnerTypesCols+" FROM partner_types ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch partner types [db fetch]", err)
	}

	defer rows.Close()

	partnerTypes, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.PartnerTypes])

	if err != nil {
		return resp.Err("Failed to fetch partner types [db fetch]", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.PartnerList{
			Partners:     partners,
			PartnerTypes: partnerTypes,
		},
	}
}
