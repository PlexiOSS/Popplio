// Package get_shop_purchases implements GET
// /{target_type}/{target_id}/shop/purchases — "Get Shop Purchases".
//
// Returns the purchase history for an entity — public, same transparency
// level as vote credit logs.
package get_shop_purchases

import (
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	shopPurchaseColsArr = dbutil.GetCols(types.ShopPurchase{})
	shopPurchaseCols    = strings.Join(shopPurchaseColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Purchases",
		Description: "Returns the purchase history for an entity.",
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
		Resp: types.ItemList[types.ShopPurchase]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetID := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetID == "" || targetType == "" {
		return resp.BadRequest("target_id and target_type are required")
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+shopPurchaseCols+" FROM shop_purchases WHERE target_id = $1 AND target_type = $2 ORDER BY created_at DESC", targetID, targetType)

	if err != nil {
		return resp.Err("Failed to fetch shop purchases", err)
	}

	defer rows.Close()

	purchases, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopPurchase])

	if err != nil {
		return resp.Err("Failed to fetch shop purchases", err)
	}

	return uapi.HttpResponse{
		Json: types.ItemList[types.ShopPurchase]{
			Items: purchases,
		},
	}
}
