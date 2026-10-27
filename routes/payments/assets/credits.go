// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"popplio/api"
	"popplio/api/resp"
	"popplio/db"
	"popplio/notifications"
	"popplio/perms"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const CreditsPerCent = 2

const MaxCreditShare = 0.5

var (
	ErrNotEnoughCredits = errors.New("not enough credits")
	ErrCreditPermission = errors.New("missing permission to spend credits")
)

func PriceCents(price float32) int {
	return int(math.Round(float64(price) * 100))
}

func CreditDiscount(priceCents, offered int) (used, discountCents int) {
	if offered <= 0 || priceCents <= 0 {
		return 0, 0
	}

	maxDiscount := int(float64(priceCents) * MaxCreditShare)
	discountCents = min(offered/CreditsPerCent, maxDiscount)

	return discountCents * CreditsPerCent, discountCents
}

func AvailableCredits(ctx context.Context, targetType, targetID string) (int, error) {
	summary, err := votes.EntityGetVoteRedeemLogsSummary(ctx, state.Pool, targetID, targetType)

	if err != nil {
		return 0, err
	}

	return summary.AvailableCredits, nil
}

func CheckCreditSpend(ctx context.Context, auth uapi.AuthData, targetType, targetID string, credits int) error {
	if err := api.AuthzEntityPermissionCheck(ctx, auth, targetType, targetID, perms.EntityBuyShopItems); err != nil {
		return ErrCreditPermission
	}

	available, err := AvailableCredits(ctx, targetType, targetID)

	if err != nil {
		return err
	}

	if available < credits {
		return fmt.Errorf("%w: has %d", ErrNotEnoughCredits, available)
	}

	return nil
}

func CreditSpendResponse(err error, targetType string) uapi.HttpResponse {
	switch {
	case errors.Is(err, ErrCreditPermission):
		return resp.Status(http.StatusForbidden, "You need permission to spend this "+targetType+"'s credits")
	case errors.Is(err, ErrNotEnoughCredits):
		return resp.BadRequest("This " + targetType + " doesn't have enough credits")
	default:
		return resp.Err("Error while checking credits", err)
	}
}

func HoldCredits(ctx context.Context, provider, externalID, userID, targetType, targetID string, credits, discountCents int) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	summary, err := votes.EntityGetVoteRedeemLogsSummary(ctx, tx, targetID, targetType)

	if err != nil {
		return err
	}

	if summary.AvailableCredits < credits {
		return ErrNotEnoughCredits
	}

	batches, err := q.GetRedeemableCreditBatches(ctx, db.GetRedeemableCreditBatchesParams{
		TargetID:   targetID,
		TargetType: targetType,
	})

	if err != nil {
		return err
	}

	remaining := credits

	for _, batch := range batches {
		if remaining <= 0 {
			break
		}

		spend := min(int(batch.Credits)-int(batch.RedeemedCredits), remaining)

		if err := q.SpendCreditBatch(ctx, db.SpendCreditBatchParams{
			RedeemedCredits: int32(spend),
			ID:              batch.ID,
		}); err != nil {
			return err
		}

		remaining -= spend
	}

	if remaining > 0 {
		return ErrNotEnoughCredits
	}

	if err := q.InsertPaymentCreditHold(ctx, db.InsertPaymentCreditHoldParams{
		Provider:      provider,
		ExternalID:    externalID,
		UserID:        userID,
		TargetType:    targetType,
		TargetID:      targetID,
		Credits:       int32(credits),
		DiscountCents: int32(discountCents),
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func RefundCreditHold(ctx context.Context, hold db.ListStalePaymentCreditHoldsRow) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	updated, err := q.MarkPaymentCreditHoldRefunded(ctx, hold.ID)

	if err != nil {
		return err
	}

	if updated == 0 {
		return nil
	}

	if _, err := q.InsertVoteRedeemLog(ctx, db.InsertVoteRedeemLogParams{
		TargetID:   hold.TargetID,
		TargetType: hold.TargetType,
		Credits:    hold.Credits,
	}); err != nil {
		return fmt.Errorf("returning credits: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if err := notifications.PushNotification(hold.UserID, types.Alert{
		Type:     types.AlertTypeInfo,
		Title:    "Credits returned",
		Message:  strconv.Itoa(int(hold.Credits)) + " credits you applied to a premium checkout were returned because the payment wasn't completed.",
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + hold.TargetType + "s/" + hold.TargetID, Valid: true},
		Category: types.AlertCategoryPayments,
	}); err != nil {
		state.Logger.Warn("Failed to send credits returned alert", zap.Error(err), zap.String("user_id", hold.UserID))
	}

	return nil
}
