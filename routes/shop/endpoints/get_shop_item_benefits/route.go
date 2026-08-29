// Package get_shop_item_benefits implements GET /shop/item-benefits — "Get
// Shop Items".
//
// Gets the publicly viewable shop items on the list
package get_shop_item_benefits

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
		Resp:        types.ItemList[types.ShopItemBenefit]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetShopItemBenefits(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch shop item benefits list [db fetch]", err)
	}

	items := make([]types.ShopItemBenefit, len(rows))
	for i, row := range rows {
		items[i] = types.ShopItemBenefit{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			CreatedAt:   row.CreatedAt.Time,
			LastUpdated: row.LastUpdated.Time,
			CreatedByID: row.CreatedBy,
			UpdatedByID: row.UpdatedBy,
			TargetTypes: row.TargetTypes,
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopItemBenefit]{
			Items: items,
		},
	}
}
