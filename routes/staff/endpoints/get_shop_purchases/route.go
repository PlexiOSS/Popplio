// Package get_shop_purchases implements GET /staff/shop-purchases —
// "Staff: Get Shop Purchases".
//
// Gets recent shop purchases across every entity, for abuse/fraud
// monitoring. The same data is also public one entity at a time via
// GET /{target_type}/{target_id}/shop/purchases; this is the platform-wide
// view staff need that endpoint can't give them.
package get_shop_purchases

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const listLimit = 100

var (
	shopPurchaseColsArr = db.GetCols(types.ShopPurchase{})
	shopPurchaseCols    = strings.Join(shopPurchaseColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Get Shop Purchases",
		Description: "Gets the most recent shop purchases across every entity.",
		Resp:        types.ItemList[types.ShopPurchase]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	staffPerms, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	if !staffPerms.Has(perms.StaffViewShop) {
		return resp.Forbidden("You do not have permission to view the shop.")
	}

	rows, err := state.Pool.Query(d.Context, "SELECT "+shopPurchaseCols+" FROM shop_purchases ORDER BY created_at DESC LIMIT "+strconv.Itoa(listLimit))

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
