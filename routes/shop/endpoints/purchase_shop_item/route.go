package purchase_shop_item

import (
	"errors"
	"math"
	"net/http"
	"slices"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/notifications"
	"popplio/routes/shop/assets"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/votes"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

type PurchaseShopItem struct {
	ItemID     string `json:"item_id" validate:"required" msg:"Item ID is required."`
	CouponCode string `json:"coupon_code" description:"An optional coupon code to apply to this purchase"`
}

var (
	compiledMessages = uapi.CompileValidationErrors(PurchaseShopItem{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Purchase Shop Item",
		Description: "Spends an entity's earned vote credits on a shop item, optionally discounted by a coupon code. Returns a 204 on success.",
		Req:         PurchaseShopItem{},
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
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetID := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetID == "" || targetType == "" {
		return resp.BadRequest("target_id and target_type are required")
	}

	if targetType != "bot" && targetType != "server" {
		return resp.BadRequest("Only bots and servers can purchase shop items")
	}

	var payload PurchaseShopItem

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errs := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errs)
	}

	itemRow, err := db.New(state.Pool).GetShopItemByID(d.Context, payload.ItemID)

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.BadRequest("Shop item not found")
	}

	if err != nil {
		return resp.Err("Failed to fetch shop item", err)
	}

	item := types.ShopItem{
		ID:          itemRow.ID,
		Name:        itemRow.Name,
		Cents:       itemRow.Cents,
		TargetTypes: itemRow.TargetTypes,
		Benefits:    itemRow.Benefits,
		CreatedAt:   itemRow.CreatedAt.Time,
		LastUpdated: itemRow.LastUpdated.Time,
		CreatedByID: itemRow.CreatedBy,
		UpdatedByID: itemRow.UpdatedBy,
		Duration:    itemRow.Duration,
		Description: itemRow.Description,
	}

	if !slices.Contains(item.TargetTypes, targetType) {
		return resp.BadRequest("This item cannot be purchased for a " + targetType)
	}

	if !assets.HasRecognizedBenefit(item.Benefits) {
		return resp.BadRequest("This item doesn't have a purchasable effect configured yet — contact staff")
	}

	itemCents := int(math.Round(item.Cents))

	var coupon *db.ShopCoupon

	if payload.CouponCode != "" {
		c, err := db.New(state.Pool).GetShopCouponByCode(d.Context, payload.CouponCode)

		if errors.Is(err, pgx.ErrNoRows) {
			return resp.BadRequest("Invalid coupon code")
		}

		if err != nil {
			return resp.Err("Failed to look up coupon", err)
		}

		if !c.Usable {
			return resp.BadRequest("This coupon is no longer usable")
		}

		if c.Expiry.Valid && time.Now().After(c.CreatedAt.Time.Add(time.Duration(c.Expiry.Int32)*time.Hour)) {
			return resp.BadRequest("This coupon has expired")
		}

		if len(c.ApplicableItems) > 0 && !slices.Contains(c.ApplicableItems, item.ID) {
			return resp.BadRequest("This coupon can't be used on this item")
		}

		if len(c.TargetTypes) > 0 && !slices.Contains(c.TargetTypes, targetType) {
			return resp.BadRequest("This coupon can't be used for a " + targetType)
		}

		if len(c.AllowedUsers) > 0 && !slices.Contains(c.AllowedUsers, d.Auth.ID) {
			return resp.BadRequest("This coupon isn't available to you")
		}

		if c.Cents.Valid {
			itemCents = int(math.Round(c.Cents.Float64))
		} else {
			itemCents = 0
		}

		coupon = &c
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.ErrBody("Error starting transaction", "An error occurred while starting transaction.", err)
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	if coupon != nil {
		if coupon.MaxUses.Valid {
			uses, err := q.CountShopCouponRedemptions(d.Context, coupon.ID)

			if err != nil {
				return resp.Err("Failed to check coupon usage", err)
			}

			if uses >= int64(coupon.MaxUses.Int32) {
				return resp.BadRequest("This coupon has reached its usage limit")
			}
		}

		if coupon.ReuseWaitDuration.Valid {
			last, err := q.GetLastShopCouponRedemptionForTarget(d.Context, db.GetLastShopCouponRedemptionForTargetParams{
				CouponID:   coupon.ID,
				TargetType: targetType,
				TargetID:   targetID,
			})

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return resp.Err("Failed to check coupon reuse cooldown", err)
			}

			if err == nil {
				waitUntil := last.Time.Add(time.Duration(coupon.ReuseWaitDuration.Int32) * time.Hour)

				if time.Now().Before(waitUntil) {
					return resp.BadRequest("This coupon can't be reused yet for this " + targetType)
				}
			}
		}
	}

	summary, err := votes.EntityGetVoteRedeemLogsSummary(d.Context, tx, targetID, targetType)

	if err != nil {
		return resp.ErrBody("An error occurred while checking available credits", "An error occurred while checking available credits.", err)
	}

	if summary.AvailableCredits < itemCents {
		return resp.BadRequest("Not enough credits to purchase this item")
	}

	batches, err := q.GetRedeemableCreditBatches(d.Context, db.GetRedeemableCreditBatchesParams{
		TargetID:   targetID,
		TargetType: targetType,
	})

	if err != nil {
		return resp.ErrBody("An error occurred while fetching credit batches", "An error occurred while fetching credit batches.", err)
	}

	remaining := itemCents

	for _, batch := range batches {
		if remaining <= 0 {
			break
		}

		available := int(batch.Credits) - int(batch.RedeemedCredits)
		spend := min(available, remaining)

		err = q.SpendCreditBatch(d.Context, db.SpendCreditBatchParams{
			RedeemedCredits: int32(spend),
			ID:              batch.ID,
		})

		if err != nil {
			return resp.ErrBody("An error occurred while spending credits", "An error occurred while spending credits.", err)
		}

		remaining -= spend
	}

	if remaining > 0 {
		return resp.ErrBody("Not enough available credits to complete this purchase", "Not enough available credits to complete this purchase.", nil)
	}

	err = q.InsertShopPurchase(d.Context, db.InsertShopPurchaseParams{
		TargetType: targetType,
		TargetID:   targetID,
		ItemID:     item.ID,
		Cents:      float64(itemCents),
	})

	if err != nil {
		return resp.ErrBody("An error occurred while logging the purchase", "An error occurred while logging the purchase.", err)
	}

	if coupon != nil {
		err = q.InsertShopCouponRedemption(d.Context, db.InsertShopCouponRedemptionParams{
			CouponID:   coupon.ID,
			TargetType: targetType,
			TargetID:   targetID,
			RedeemedBy: d.Auth.ID,
		})

		if err != nil {
			return resp.ErrBody("An error occurred while logging the coupon redemption", "An error occurred while logging the coupon redemption.", err)
		}
	}

	for _, benefitID := range item.Benefits {
		if err := assets.ApplyBenefit(d.Context, tx, benefitID, targetType, targetID, item.Duration); err != nil {
			return resp.ErrBody("An error occurred while applying the purchase", "An error occurred while applying the purchase.", err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return resp.ErrBody("Error committing transaction", "An error occurred while committing transaction.", err)
	}

	if err := notifications.PushNotification(d.Auth.ID, types.Alert{
		Type:     types.AlertTypeSuccess,
		Title:    "Shop Purchase Complete",
		Message:  item.Name + " has been applied to " + targetID + ".",
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + targetType + "s/" + targetID, Valid: true},
		Category: types.AlertCategoryShop,
	}); err != nil {
		state.Logger.Warn("Failed to send shop purchase confirmation alert", zap.Error(err), zap.String("user_id", d.Auth.ID), zap.String("item_id", item.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
