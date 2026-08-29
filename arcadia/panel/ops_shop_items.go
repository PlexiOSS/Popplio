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
)

func validateShopItem(ctx context.Context, action *types.ShopItemUpsert) (*response, error) {
	if action.Cents < 0 {
		resp := writeText(http.StatusBadRequest, "Cents cannot be lower than 0")
		return &resp, nil
	}

	if action.Duration < 0 {
		resp := writeText(http.StatusBadRequest, "Duration cannot be lower than 0")
		return &resp, nil
	}

	q := db.New(state.Pool)

	for _, benefit := range action.Benefits {
		exists, err := q.CountShopItemBenefitByID(ctx, benefit)

		if err != nil {
			return nil, err
		}

		if !exists {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf("Benefit %s does not exist", benefit))
			return &resp, nil
		}
	}

	return nil, nil
}

func (s *Server) updateShopItems(ctx context.Context, q *types.QUpdateShopItems) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		itemRows, err := db.New(state.Pool).GetShopItems(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		items := make([]types.ShopItem, 0, len(itemRows))

		for _, i := range itemRows {
			items = append(items, types.ShopItem{
				ID:          i.ID,
				Name:        i.Name,
				Description: i.Description,
				Cents:       i.Cents,
				TargetTypes: types.NonNilStrings(i.TargetTypes),
				Benefits:    types.NonNilStrings(i.Benefits),
				Duration:    int32(i.Duration),
				CreatedAt:   types.NewTimestamp(i.CreatedAt.Time),
				LastUpdated: types.NewTimestamp(i.LastUpdated.Time),
				CreatedBy:   i.CreatedBy,
				UpdatedBy:   i.UpdatedBy,
			})
		}

		return writeJSON(http.StatusOK, items), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create shop items [manage_shop]"), nil
		}

		resp, err := validateShopItem(ctx, action)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		err = db.New(state.Pool).InsertShopItem(ctx, db.InsertShopItemParams{
			ID:          action.ID,
			Name:        action.Name,
			Cents:       action.Cents,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			Benefits:    types.NonNilStrings(action.Benefits),
			CreatedBy:   authData.UserID,
			Duration:    int64(action.Duration),
			Description: action.Description,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update shop items [manage_shop]"), nil
		}

		resp, err := validateShopItem(ctx, action)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		exists, err := db.New(state.Pool).CountShopItemByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = db.New(state.Pool).UpdateShopItem(ctx, db.UpdateShopItemParams{
			Name:        action.Name,
			Cents:       action.Cents,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			Benefits:    types.NonNilStrings(action.Benefits),
			UpdatedBy:   authData.UserID,
			Duration:    int64(action.Duration),
			Description: action.Description,
			ID:          action.ID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete shop items [manage_shop]"), nil
		}

		id := q.Action.Delete.ID

		exists, err := db.New(state.Pool).CountShopItemByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := db.New(state.Pool).DeleteShopItem(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No shop item action was specified")
	}
}
