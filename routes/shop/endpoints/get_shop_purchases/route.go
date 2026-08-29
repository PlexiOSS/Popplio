package get_shop_purchases

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
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

	rows, err := db.New(state.Pool).GetShopPurchasesByTarget(d.Context, db.GetShopPurchasesByTargetParams{
		TargetID:   targetID,
		TargetType: targetType,
	})

	if err != nil {
		return resp.Err("Failed to fetch shop purchases", err)
	}

	purchases := make([]types.ShopPurchase, len(rows))
	for i, row := range rows {
		purchases[i] = types.ShopPurchase{
			ID:         row.ID,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			ItemID:     row.ItemID,
			Cents:      row.Cents,
			CreatedAt:  row.CreatedAt.Time,
		}
	}

	return uapi.HttpResponse{
		Json: types.ItemList[types.ShopPurchase]{
			Items: purchases,
		},
	}
}
