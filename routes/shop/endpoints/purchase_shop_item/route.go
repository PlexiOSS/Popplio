// Package purchase_shop_item implements POST
// /{target_type}/{target_id}/shop/purchase — "Purchase Shop Item".
//
// Spends an entity's earned vote credits on a shop item, applying whatever
// recognized benefits the item grants. Returns a 204 on success.
package purchase_shop_item

import (
	"errors"
	"math"
	"net/http"
	"slices"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
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
	ItemID string `json:"item_id" validate:"required" msg:"Item ID is required."`
}

var (
	compiledMessages = uapi.CompileValidationErrors(PurchaseShopItem{})

	shopItemColsArr = dbutil.GetCols(types.ShopItem{})
	shopItemCols    = strings.Join(shopItemColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Purchase Shop Item",
		Description: "Spends an entity's earned vote credits on a shop item. Returns a 204 on success.",
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

type redeemLogRow struct {
	ID              pgtype.UUID `db:"id"`
	Credits         int         `db:"credits"`
	RedeemedCredits int         `db:"redeemed_credits"`
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

	itemRows, err := state.Pool.Query(d.Context, "SELECT "+shopItemCols+" FROM shop_items WHERE id = $1", payload.ItemID)

	if err != nil {
		return resp.Err("Failed to fetch shop item", err)
	}

	item, err := pgx.CollectOneRow(itemRows, pgx.RowToStructByName[types.ShopItem])

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.BadRequest("Shop item not found")
	}

	if err != nil {
		return resp.Err("Failed to fetch shop item", err)
	}

	if !slices.Contains(item.TargetTypes, targetType) {
		return resp.BadRequest("This item cannot be purchased for a " + targetType)
	}

	if !assets.HasRecognizedBenefit(item.Benefits) {
		return resp.BadRequest("This item doesn't have a purchasable effect configured yet — contact staff")
	}

	itemCents := int(math.Round(item.Cents))

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.ErrBody("Error starting transaction", "An error occurred while starting transaction.", err)
	}

	defer tx.Rollback(d.Context)

	summary, err := votes.EntityGetVoteRedeemLogsSummary(d.Context, tx, targetID, targetType)

	if err != nil {
		return resp.ErrBody("An error occurred while checking available credits", "An error occurred while checking available credits.", err)
	}

	if summary.AvailableCredits < itemCents {
		return resp.BadRequest("Not enough credits to purchase this item")
	}

	// Spend oldest-first across every not-fully-spent credit batch this
	// entity has earned, locking the rows so a concurrent purchase can't
	// double-spend the same credits.
	logRows, err := tx.Query(d.Context,
		"SELECT id, credits, redeemed_credits FROM entity_vote_redeem_logs WHERE target_id = $1 AND target_type = $2 AND redeemed_credits < credits ORDER BY created_at ASC FOR UPDATE",
		targetID, targetType,
	)

	if err != nil {
		return resp.ErrBody("An error occurred while fetching credit batches", "An error occurred while fetching credit batches.", err)
	}

	batches, err := pgx.CollectRows(logRows, pgx.RowToStructByName[redeemLogRow])

	if err != nil {
		return resp.ErrBody("An error occurred while collecting credit batches", "An error occurred while collecting credit batches.", err)
	}

	remaining := itemCents

	for _, batch := range batches {
		if remaining <= 0 {
			break
		}

		available := batch.Credits - batch.RedeemedCredits
		spend := min(available, remaining)

		_, err = tx.Exec(d.Context,
			"UPDATE entity_vote_redeem_logs SET redeemed_credits = redeemed_credits + $1, redeemed_at = NOW() WHERE id = $2",
			spend, batch.ID,
		)

		if err != nil {
			return resp.ErrBody("An error occurred while spending credits", "An error occurred while spending credits.", err)
		}

		remaining -= spend
	}

	if remaining > 0 {
		return resp.ErrBody("Not enough available credits to complete this purchase", "Not enough available credits to complete this purchase.", nil)
	}

	_, err = tx.Exec(d.Context,
		"INSERT INTO shop_purchases (target_type, target_id, item_id, cents) VALUES ($1, $2, $3, $4)",
		targetType, targetID, item.ID, item.Cents,
	)

	if err != nil {
		return resp.ErrBody("An error occurred while logging the purchase", "An error occurred while logging the purchase.", err)
	}

	for _, benefitID := range item.Benefits {
		if err := assets.ApplyBenefit(d.Context, tx, benefitID, targetType, targetID, item.Duration); err != nil {
			return resp.ErrBody("An error occurred while applying the purchase", "An error occurred while applying the purchase.", err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return resp.ErrBody("Error committing transaction", "An error occurred while committing transaction.", err)
	}

	// Best-effort: the purchase already committed, so a failure here
	// shouldn't turn into an error response the client would read as "the
	// purchase failed" when it didn't.
	if err := notifications.PushNotification(d.Auth.ID, types.Alert{
		Type:    types.AlertTypeSuccess,
		Title:   "Shop Purchase Complete",
		Message: item.Name + " has been applied to " + targetID + ".",
	}); err != nil {
		state.Logger.Warn("Failed to send shop purchase confirmation alert", zap.Error(err), zap.String("user_id", d.Auth.ID), zap.String("item_id", item.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
