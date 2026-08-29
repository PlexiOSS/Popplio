// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"fmt"
	"net/http"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

func validateCoupon(action *types.ShopCouponUpsert) *response {
	if action.MaxUses != nil && *action.MaxUses <= 0 {
		resp := writeText(http.StatusBadRequest, "Max uses must be greater than 0")
		return &resp
	}

	if action.ReuseWaitDuration != nil && *action.ReuseWaitDuration <= 0 {
		resp := writeText(http.StatusBadRequest, "Reuse wait duration must be greater than 0")
		return &resp
	}

	if action.Expiry != nil && *action.Expiry <= 0 {
		resp := writeText(http.StatusBadRequest, "Expiry must be greater than 0")
		return &resp
	}

	if action.Cents != nil && *action.Cents < 0 {
		resp := writeText(http.StatusBadRequest, "Cents cannot be lower than 0")
		return &resp
	}

	return nil
}

func validateCouponItems(ctx context.Context, items []string) (*response, error) {
	q := db.New(state.Pool)

	for _, item := range items {
		exists, err := q.CountShopItemByID(ctx, item)

		if err != nil {
			return nil, err
		}

		if !exists {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf("Item %q does not exist", item))
			return &resp, nil
		}
	}

	return nil, nil
}

func int4FromPtr(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{Int32: *v, Valid: true}
}

func float8FromPtr(v *float64) pgtype.Float8 {
	if v == nil {
		return pgtype.Float8{}
	}

	return pgtype.Float8{Float64: *v, Valid: true}
}

func ptrFromInt4(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}

	return &v.Int32
}

func ptrFromFloat8(v pgtype.Float8) *float64 {
	if !v.Valid {
		return nil
	}

	return &v.Float64
}

func (s *Server) updateShopCoupons(ctx context.Context, q *types.QUpdateShopCoupons) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		if !userPerms.Has(perms.StaffViewShop) {
			return writeText(http.StatusForbidden, "You do not have permission to list shop coupons [view_shop]"), nil
		}

		couponRows, err := db.New(state.Pool).ListShopCoupons(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		coupons := make([]types.ShopCoupon, 0, len(couponRows))

		for _, c := range couponRows {
			coupons = append(coupons, types.ShopCoupon{
				ID:                c.ID,
				Code:              c.Code,
				Public:            c.Public,
				MaxUses:           ptrFromInt4(c.MaxUses),
				CreatedAt:         types.NewTimestamp(c.CreatedAt.Time),
				CreatedBy:         c.CreatedBy,
				LastUpdated:       types.NewTimestamp(c.LastUpdated.Time),
				UpdatedBy:         c.UpdatedBy,
				ReuseWaitDuration: ptrFromInt4(c.ReuseWaitDuration),
				Expiry:            ptrFromInt4(c.Expiry),
				ApplicableItems:   types.NonNilStrings(c.ApplicableItems),
				Cents:             ptrFromFloat8(c.Cents),
				Requirements:      types.NonNilStrings(c.Requirements),
				AllowedUsers:      types.NonNilStrings(c.AllowedUsers),
				Usable:            c.Usable,
				TargetTypes:       types.NonNilStrings(c.TargetTypes),
			})
		}

		return writeJSON(http.StatusOK, coupons), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create shop coupons [manage_shop]"), nil
		}

		if resp := validateCoupon(action); resp != nil {
			return *resp, nil
		}

		resp, err := validateCouponItems(ctx, action.ApplicableItems)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		err = db.New(state.Pool).InsertShopCoupon(ctx, db.InsertShopCouponParams{
			ID:                action.ID,
			Code:              action.Code,
			Public:            action.Public,
			MaxUses:           int4FromPtr(action.MaxUses),
			CreatedBy:         authData.UserID,
			ReuseWaitDuration: int4FromPtr(action.ReuseWaitDuration),
			Expiry:            int4FromPtr(action.Expiry),
			ApplicableItems:   types.NonNilStrings(action.ApplicableItems),
			Cents:             float8FromPtr(action.Cents),
			Requirements:      types.NonNilStrings(action.Requirements),
			AllowedUsers:      types.NonNilStrings(action.AllowedUsers),
			Usable:            action.Usable,
			TargetTypes:       types.NonNilStrings(action.TargetTypes),
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update shop coupons [manage_shop]"), nil
		}

		if resp := validateCoupon(action); resp != nil {
			return *resp, nil
		}

		resp, err := validateCouponItems(ctx, action.ApplicableItems)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		queries := db.New(state.Pool)

		exists, err := queries.CountShopCouponByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = queries.UpdateShopCoupon(ctx, db.UpdateShopCouponParams{
			Code:              action.Code,
			Public:            action.Public,
			MaxUses:           int4FromPtr(action.MaxUses),
			ReuseWaitDuration: int4FromPtr(action.ReuseWaitDuration),
			Expiry:            int4FromPtr(action.Expiry),
			ApplicableItems:   types.NonNilStrings(action.ApplicableItems),
			Cents:             float8FromPtr(action.Cents),
			Requirements:      types.NonNilStrings(action.Requirements),
			UpdatedBy:         authData.UserID,
			AllowedUsers:      types.NonNilStrings(action.AllowedUsers),
			Usable:            action.Usable,
			TargetTypes:       types.NonNilStrings(action.TargetTypes),
			ID:                action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete shop coupons [manage_shop]"), nil
		}

		id := q.Action.Delete.ID

		queries := db.New(state.Pool)

		exists, err := queries.CountShopCouponByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := queries.DeleteShopCoupon(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No shop coupon action was specified")
	}
}
