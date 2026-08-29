package get_public_coupons

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func intPtr(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int32)
	return &i
}

func float64Ptr(v pgtype.Float8) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Shop Coupons",
		Description: "Gets the publicly viewable shop coupons on the list",
		Resp:        types.ItemList[types.ShopCoupon]{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetPublicShopCoupons(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch shop coupons list [db fetch]", err)
	}

	coupons := make([]types.ShopCoupon, len(rows))
	for i, row := range rows {
		coupons[i] = types.ShopCoupon{
			ID:                row.ID,
			Code:              row.Code,
			Public:            row.Public,
			MaxUses:           intPtr(row.MaxUses),
			Cents:             float64Ptr(row.Cents),
			Requirements:      row.Requirements,
			AllowedUsers:      row.AllowedUsers,
			CreatedAt:         row.CreatedAt.Time,
			LastUpdated:       row.LastUpdated.Time,
			CreatedByID:       row.CreatedBy,
			UpdatedByID:       row.UpdatedBy,
			ReuseWaitDuration: intPtr(row.ReuseWaitDuration),
			Expiry:            intPtr(row.Expiry),
			ApplicableItems:   row.ApplicableItems,
			Usable:            row.Usable,
			TargetTypes:       row.TargetTypes,
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.ItemList[types.ShopCoupon]{
			Items: coupons,
		},
	}
}
