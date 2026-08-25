// Package get_shop_items implements GET /shop/items — "Get Shop Items".
//
// Gets the publicly viewable shop items on the list
package get_shop_items

import (
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	// Shop items
	shopItemsColsArr = dbutil.GetCols(types.ShopItem{})
	shopItemsCols    = strings.Join(shopItemsColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Items",
		Description: "Gets the publicly viewable shop items on the list",
		Resp:        types.ItemList[types.ShopItem]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+shopItemsCols+" FROM shop_items ORDER BY created_at DESC")

	if err != nil {
		return resp.Err("Failed to fetch shop items list [db fetch]", err)
	}

	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ShopItem])

	if err != nil {
		return resp.Err("Failed to fetch shop items list [db fetch]", err)
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopItem]{
			Items: items,
		},
	}
}
