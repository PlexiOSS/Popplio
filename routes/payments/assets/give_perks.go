package assets

import (
	"context"
	"errors"
	"strconv"

	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type CreatePerkData struct {
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
	ForType     string `json:"for_type" validate:"omitempty,oneof=bot server" msg:"for_type must be 'bot' or 'server' if sent."`
}

type RedirectUser struct {
	URL string `json:"url"`
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

func GivePerks(ctx context.Context, perkData PerkData) error {
	perk, err := FindPerks(ctx, perkData)

	if err != nil {
		return err
	}

	switch perkData.ProductID {
	case "premium":
		targetID := perkData.For

		q := db.New(state.Pool)

		switch perkData.ForType {
		case "bot":
			err = q.ApplyBotPremiumDays(ctx, db.ApplyBotPremiumDaysParams{Hours: int32(perk.TimePeriod), BotID: targetID})
		case "server":
			err = q.ApplyServerPremiumDays(ctx, db.ApplyServerPremiumDaysParams{Hours: int32(perk.TimePeriod), ServerID: targetID})
		default:
			return errors.New("invalid for_type")
		}

		if err != nil {
			return errors.New("our database broke, please try again later")
		}

		mention := targetID
		if perkData.ForType == "bot" {
			mention = "<@" + targetID + ">"
		}

		_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
			Content: "<@" + perkData.UserID + "> has bought " + mention + " (" + perkData.ForType + ") premium for " + strconv.Itoa(perk.TimePeriod) + " hours.",
		})

		if err != nil {
			return errors.New("couldn't send message to mod logs")
		}

		if err := notifications.PushNotification(perkData.UserID, types.Alert{
			Type:     types.AlertTypeSuccess,
			Title:    "Premium Activated",
			Message:  perkData.For + " (" + perkData.ForType + ") now has premium for the next " + strconv.Itoa(perk.TimePeriod) + " hours.",
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + perkData.ForType + "s/" + perkData.For, Valid: true},
			Category: types.AlertCategoryPayments,
		}); err != nil {
			state.Logger.Warn("Failed to send premium confirmation alert", zap.Error(err), zap.String("user_id", perkData.UserID), zap.String("for", targetID))
		}
	}

	return nil
}
