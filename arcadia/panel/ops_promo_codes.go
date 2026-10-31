// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"errors"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/promo"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

func intFromInt4(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}

	i := int(v.Int32)
	return &i
}

func (s *Server) updatePromoCodes(ctx context.Context, q *types.QUpdatePromoCodes) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		if !userPerms.Has(perms.StaffViewShop) {
			return writeText(http.StatusForbidden, "You do not have permission to list promo codes [view_shop]"), nil
		}

		rows, err := db.New(state.Pool).ListPromoCodes(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		codes := make([]types.PromoCode, 0, len(rows))

		for _, c := range rows {
			var expires *types.Timestamp

			if c.ExpiresAt.Valid {
				ts := types.NewTimestamp(c.ExpiresAt.Time)
				expires = &ts
			}

			codes = append(codes, types.PromoCode{
				ID:                    impls.UUIDString(c.ID),
				Code:                  c.Code,
				Description:           c.Description,
				Products:              types.NonNilStrings(c.Products),
				PercentOff:            intFromInt4(c.PercentOff),
				AmountOffCents:        intFromInt4(c.AmountOffCents),
				Duration:              c.Duration,
				DurationMonths:        intFromInt4(c.DurationMonths),
				MaxRedemptions:        intFromInt4(c.MaxRedemptions),
				Redemptions:           int(c.Redemptions),
				ExpiresAt:             expires,
				OncePerUser:           c.OncePerUser,
				NewCustomersOnly:      c.NewCustomersOnly,
				Active:                c.Active,
				StripeCouponID:        c.StripeCouponID,
				StripePromotionCodeID: c.StripePromotionCodeID,
				CreatedBy:             c.CreatedBy,
				CreatedAt:             types.NewTimestamp(c.CreatedAt.Time),
				UpdatedAt:             types.NewTimestamp(c.UpdatedAt.Time),
			})
		}

		return writeJSON(http.StatusOK, codes), nil
	case q.Action.Create != nil:
		if !userPerms.Has(perms.StaffManagePromoCodes) {
			return writeText(http.StatusForbidden, "You do not have permission to create promo codes [manage_promo_codes]"), nil
		}

		action := q.Action.Create

		var expires *time.Time

		if action.ExpiresAt != nil {
			expires = &action.ExpiresAt.Time
		}

		id, err := promo.Create(ctx, promo.CreateParams{
			Code:             action.Code,
			Description:      action.Description,
			Products:         action.Products,
			PercentOff:       action.PercentOff,
			AmountOffCents:   action.AmountOffCents,
			Duration:         action.Duration,
			DurationMonths:   action.DurationMonths,
			MaxRedemptions:   action.MaxRedemptions,
			ExpiresAt:        expires,
			OncePerUser:      action.OncePerUser,
			NewCustomersOnly: action.NewCustomersOnly,
		}, authData.UserID)

		if resp, ok := promoRejection(err); ok {
			return resp, nil
		}

		if err != nil {
			return response{}, newError(err)
		}

		return writeJSON(http.StatusOK, map[string]string{"id": impls.UUIDString(id)}), nil
	case q.Action.SetActive != nil:
		if !userPerms.Has(perms.StaffManagePromoCodes) {
			return writeText(http.StatusForbidden, "You do not have permission to change promo codes [manage_promo_codes]"), nil
		}

		var id pgtype.UUID

		if err := id.Scan(q.Action.SetActive.ID); err != nil {
			return writeText(http.StatusBadRequest, "Invalid promo code ID"), nil
		}

		err := promo.SetActive(ctx, id, q.Action.SetActive.Active)

		if resp, ok := promoRejection(err); ok {
			return resp, nil
		}

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No promo code action was specified")
	}
}

func promoRejection(err error) (response, bool) {
	var rej *promo.Rejection

	if errors.As(err, &rej) {
		return writeText(http.StatusBadRequest, rej.Reason), true
	}

	return response{}, false
}
