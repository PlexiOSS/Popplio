// Package get_shop_items implements GET /shop/items — "Get Shop Items".
//
// Gets the publicly viewable shop items on the list
package get_shop_items

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Items",
		Description: "Gets the publicly viewable shop items on the list",
		Resp:        types.ItemList[types.ShopItem]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetShopItems(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch shop items list [db fetch]", err)
	}

	items := make([]types.ShopItem, len(rows))
	for i, row := range rows {
		items[i] = types.ShopItem{
			ID:          row.ID,
			Name:        row.Name,
			Cents:       row.Cents,
			TargetTypes: row.TargetTypes,
			Benefits:    row.Benefits,
			CreatedAt:   row.CreatedAt.Time,
			LastUpdated: row.LastUpdated.Time,
			CreatedByID: row.CreatedBy,
			UpdatedByID: row.UpdatedBy,
			Duration:    row.Duration,
			Description: row.Description,
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopItem]{
			Items: items,
		},
	}
}
