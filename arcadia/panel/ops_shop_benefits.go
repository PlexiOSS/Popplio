// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
)

func (s *Server) updateShopItemBenefits(ctx context.Context, q *types.QUpdateShopItemBenefits) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		benefitRows, err := db.New(state.Pool).GetShopItemBenefits(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		benefits := make([]types.ShopItemBenefit, 0, len(benefitRows))

		for _, b := range benefitRows {
			benefits = append(benefits, types.ShopItemBenefit{
				ID:          b.ID,
				Name:        b.Name,
				Description: b.Description,
				CreatedAt:   types.NewTimestamp(b.CreatedAt.Time),
				LastUpdated: types.NewTimestamp(b.LastUpdated.Time),
				TargetTypes: types.NonNilStrings(b.TargetTypes),
				CreatedBy:   b.CreatedBy,
				UpdatedBy:   b.UpdatedBy,
			})
		}

		return writeJSON(http.StatusOK, benefits), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create shop item benefits [manage_shop]"), nil
		}

		err := db.New(state.Pool).InsertShopItemBenefit(ctx, db.InsertShopItemBenefitParams{
			ID:          action.ID,
			Name:        action.Name,
			Description: action.Description,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			CreatedBy:   authData.UserID,
			UpdatedBy:   authData.UserID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update shop item benefits [manage_shop]"), nil
		}

		exists, err := db.New(state.Pool).CountShopItemBenefitByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = db.New(state.Pool).UpdateShopItemBenefit(ctx, db.UpdateShopItemBenefitParams{
			Name:        action.Name,
			Description: action.Description,
			UpdatedBy:   authData.UserID,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			ID:          action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete shop item benefits [manage_shop]"), nil
		}

		id := q.Action.Delete.ID

		queries := db.New(state.Pool)

		exists, err := queries.CountShopItemBenefitByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		inUse, err := queries.CountShopItemsUsingBenefit(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if inUse {
			return writeText(http.StatusBadRequest, "Cannot delete benefit as it is used by shop items"), nil
		}

		if err := queries.DeleteShopItemBenefit(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No shop item benefit action was specified")
	}
}
