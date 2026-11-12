package assets

import (
	"context"
	"errors"
	"fmt"
	"popplio/audit"
	"popplio/brandemoji"
	"strconv"
	"strings"

	"popplio/arcadia/impls"
	arcadiatypes "popplio/arcadia/types"
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type CreatePerkData struct {
	ProductName   string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID     string `json:"id" validate:"required" msg:"Product ID is required."`
	For           string `json:"for" validate:"required" msg:"For is required."`
	ForType       string `json:"for_type" validate:"omitempty,oneof=bot server" msg:"for_type must be 'bot' or 'server' if sent."`
	Credits       int    `json:"credits" validate:"omitempty,min=0" msg:"credits can't be negative."`
	PromoCode     string `json:"promo_code" validate:"omitempty,max=40" msg:"promo_code is too long."`
	Gift          bool   `json:"gift" description:"Whether this is a gift for a listing the buyer doesn't own"`
	GiftMessage   string `json:"gift_message" validate:"omitempty,max=300" msg:"The gift message can be at most 300 characters."`
	GiftAnonymous bool   `json:"gift_anonymous" description:"Hide the buyer's name from the listing's owners"`
}

type RedirectUser struct {
	URL string `json:"url"`
}

func RecordGift(ctx context.Context, provider, externalID, gifterID string, c CreatePerkData) {
	if !c.Gift {
		return
	}

	if err := db.New(state.Pool).InsertPaymentGift(ctx, db.InsertPaymentGiftParams{
		Provider:   provider,
		ExternalID: externalID,
		GifterID:   gifterID,
		Message:    strings.TrimSpace(c.GiftMessage),
		Anonymous:  c.GiftAnonymous,
	}); err != nil {
		state.Logger.Error("Failed to record a premium gift", zap.Error(err), zap.String("provider", provider), zap.String("external_id", externalID))
	}
}

func notifyGift(ctx context.Context, provider, externalID string, perkData PerkData, hours int) {
	gift, err := db.New(state.Pool).GetPaymentGift(ctx, db.GetPaymentGiftParams{Provider: provider, ExternalID: externalID})

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			state.Logger.Warn("Failed to look up a premium gift", zap.Error(err), zap.String("external_id", externalID))
		}

		return
	}

	listing := "your " + perkData.ForType

	if info, err := votes.GetEntityInfo(ctx, state.Pool, perkData.For, perkData.ForType); err == nil {
		listing = info.Name
	}

	from := "Someone"

	if !gift.Anonymous {
		if user, err := dovewing.GetUser(ctx, gift.GifterID, state.DovewingPlatformDiscord); err == nil {
			from = user.Username

			if user.DisplayName != "" {
				from = user.DisplayName
			}
		}
	}

	message := fmt.Sprintf("%s gifted %s %s of premium.", from, listing, premiumLength(hours))

	if gift.Message != "" {
		message += " Their message: \"" + gift.Message + "\""
	}

	targetType := arcadiatypes.TargetTypeBot

	if perkData.ForType == "server" {
		targetType = arcadiatypes.TargetTypeServer
	}

	managers, err := impls.GetEntityManagers(ctx, targetType, perkData.For)

	if err != nil {
		state.Logger.Warn("Failed to find owners for a premium gift", zap.Error(err), zap.String("for", perkData.For))
		return
	}

	for _, owner := range managers.All() {
		if owner == gift.GifterID {
			continue
		}

		if err := notifications.PushNotification(owner, types.Alert{
			Type:     types.AlertTypeSuccess,
			Title:    listing + " was gifted premium",
			Message:  message,
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + perkData.ForType + "s/" + perkData.For, Valid: true},
			Category: types.AlertCategoryPayments,
		}); err != nil {
			state.Logger.Warn("Failed to notify an owner about a premium gift", zap.Error(err), zap.String("owner", owner))
		}
	}
}

func premiumLength(hours int) string {
	if hours%24 == 0 {
		days := hours / 24

		if days == 1 {
			return "1 day"
		}

		return strconv.Itoa(days) + " days"
	}

	return strconv.Itoa(hours) + " hours"
}

func (c CreatePerkData) Parse(userID string) PerkData {
	forType := c.ForType
	if forType == "" {
		forType = "bot"
	}

	return PerkData{
		UserID:      userID,
		ProductName: c.ProductName,
		ProductID:   c.ProductID,
		For:         c.For,
		ForType:     forType,
	}
}

type PerkData struct {
	UserID      string `json:"user_id" validate:"required" msg:"Internal error: endpoint must fill in UserID. Please contact support."`
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
	ForType     string `json:"for_type" validate:"required,oneof=bot server" msg:"for_type must be 'bot' or 'server'."`
}

func FindPerks(ctx context.Context, payload PerkData) (*types.PaymentPlan, error) {
	var perk *types.PaymentPlan

	state.Logger.Debug("Got payload", zap.Any("payload", payload))

	if payload.UserID == "" {
		return nil, errors.New("internal error: user id is required")
	}

	entityLabel, err := entityLabel(payload.ForType)

	if err != nil {
		return nil, err
	}

	q := db.New(state.Pool)

	switch payload.ProductID {
	case "premium":
		for _, plan := range Plans {
			if plan.ID == payload.ProductName {
				var count int64
				var typeStr string
				var premium bool

				switch payload.ForType {
				case "bot":
					count, err = q.CountBotByID(ctx, payload.For)
				case "server":
					count, err = q.CountServerByID(ctx, payload.For)
				}

				if err != nil {
					return nil, errors.New("our database broke, please try again later")
				}

				if count == 0 {
					return nil, errors.New(entityLabel + " id is invalid")
				}

				switch payload.ForType {
				case "bot":
					row, rowErr := q.GetBotTypePremium(ctx, payload.For)
					typeStr, premium, err = row.Type, row.Premium, rowErr
				case "server":
					row, rowErr := q.GetServerTypePremium(ctx, payload.For)
					typeStr, premium, err = row.Type, row.Premium, rowErr
				}

				if err != nil {
					return nil, errors.New("our database broke, please try again later")
				}

				if typeStr != "approved" && typeStr != "certified" {
					return nil, errors.New(entityLabel + " is not approved or certified")
				}

				if premium {
					return nil, errors.New(entityLabel + " is already premium")
				}

				perk = &plan

				break
			}
		}
	default:
		return nil, errors.New("invalid product id")
	}

	if perk == nil {
		return nil, errors.New("product not found")
	}

	return perk, nil
}

func entityLabel(forType string) (label string, err error) {
	switch forType {
	case "bot":
		return "bot", nil
	case "server":
		return "server", nil
	default:
		return "", errors.New("invalid for_type")
	}
}

var ErrInvalidPurchase = errors.New("invalid purchase")

func GivePerks(ctx context.Context, perkData PerkData) error {
	_, err := fulfill(ctx, "", "", perkData)
	return err
}

func Fulfill(ctx context.Context, provider, externalID string, perkData PerkData) (bool, error) {
	if provider == "" || externalID == "" {
		return false, errors.New("provider and external id are required to fulfill a payment")
	}

	fulfilled, err := fulfill(ctx, provider, externalID, perkData)

	if fulfilled {
		audit.Record(audit.Event{
			Source:     provider,
			ActorType:  audit.ActorUser,
			ActorID:    perkData.UserID,
			Action:     "payment.fulfilled",
			TargetType: perkData.ForType,
			TargetID:   perkData.For,
			Data:       map[string]any{"external_id": externalID, "product": perkData.ProductID, "plan": perkData.ProductName},
		})
	}

	return fulfilled, err
}

func fulfill(ctx context.Context, provider, externalID string, perkData PerkData) (bool, error) {
	if provider != "" {
		done, err := db.New(state.Pool).PaymentFulfillmentExists(ctx, db.PaymentFulfillmentExistsParams{Provider: provider, ExternalID: externalID})

		if err != nil {
			return false, fmt.Errorf("checking fulfillment: %w", err)
		}

		if done {
			return false, nil
		}
	}

	perk, err := FindPerks(ctx, perkData)

	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidPurchase, err)
	}

	if perkData.ProductID != "premium" {
		return false, fmt.Errorf("%w: unsupported product %q", ErrInvalidPurchase, perkData.ProductID)
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return false, fmt.Errorf("starting transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	if provider != "" {
		inserted, err := q.InsertPaymentFulfillment(ctx, db.InsertPaymentFulfillmentParams{
			Provider:     provider,
			ExternalID:   externalID,
			UserID:       perkData.UserID,
			ProductID:    perkData.ProductID,
			ForType:      perkData.ForType,
			ForID:        perkData.For,
			PremiumHours: pgtype.Int4{Int32: int32(perk.TimePeriod), Valid: true},
		})

		if err != nil {
			return false, fmt.Errorf("recording fulfillment: %w", err)
		}

		if inserted == 0 {
			return false, nil
		}

		if _, err := q.MarkPaymentCreditHoldSpent(ctx, db.MarkPaymentCreditHoldSpentParams{
			Provider:   provider,
			ExternalID: externalID,
		}); err != nil {
			return false, fmt.Errorf("settling credit hold: %w", err)
		}
	}

	switch perkData.ForType {
	case "bot":
		err = q.ApplyBotPremiumDays(ctx, db.ApplyBotPremiumDaysParams{Hours: int32(perk.TimePeriod), BotID: perkData.For})
	case "server":
		err = q.ApplyServerPremiumDays(ctx, db.ApplyServerPremiumDaysParams{Hours: int32(perk.TimePeriod), ServerID: perkData.For})
	default:
		return false, fmt.Errorf("%w: invalid for_type", ErrInvalidPurchase)
	}

	if err != nil {
		return false, fmt.Errorf("applying premium: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("committing fulfillment: %w", err)
	}

	afterFulfill(ctx, provider, externalID, perkData, perk.TimePeriod)

	return true, nil
}

var afterFulfill = func(ctx context.Context, provider, externalID string, perkData PerkData, hours int) {
	announcePremium(perkData, hours)

	if provider != "" {
		notifyGift(ctx, provider, externalID, perkData, hours)
	}
}

func announcePremium(perkData PerkData, hours int) {
	mention := perkData.For
	if perkData.ForType == "bot" {
		mention = "<@" + perkData.For + ">"
	}

	if _, err := state.Discord.Rest.CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: brandemoji.Popplio.Get("premium", "💎") + " <@" + perkData.UserID + "> has bought " + mention + " (" + perkData.ForType + ") premium for " + strconv.Itoa(hours) + " hours.",
	}); err != nil {
		state.Logger.Warn("Failed to announce premium purchase in mod logs", zap.Error(err), zap.String("user_id", perkData.UserID), zap.String("for", perkData.For))
	}

	if err := notifications.PushNotification(perkData.UserID, types.Alert{
		Type:     types.AlertTypeSuccess,
		Title:    "Premium Activated",
		Message:  perkData.For + " (" + perkData.ForType + ") now has premium for the next " + strconv.Itoa(hours) + " hours.",
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + perkData.ForType + "s/" + perkData.For, Valid: true},
		Category: types.AlertCategoryPayments,
	}); err != nil {
		state.Logger.Warn("Failed to send premium confirmation alert", zap.Error(err), zap.String("user_id", perkData.UserID), zap.String("for", perkData.For))
	}
}
