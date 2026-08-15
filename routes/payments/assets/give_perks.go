package assets

import (
	"context"
	"errors"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
)

type CreatePerkData struct {
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
	// ForType is "bot" or "server". Omitted/empty defaults to "bot" for
	// backward compatibility with clients that predate server premium.
	ForType string `json:"for_type" validate:"omitempty,oneof=bot server" msg:"for_type must be 'bot' or 'server' if sent."`
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

// Finds and validates the associated perm for the given payload. ProductID is still needed to determine whats being purchased.
func FindPerks(ctx context.Context, payload PerkData) (*types.PaymentPlan, error) {
	var perk *types.PaymentPlan

	state.Logger.Debug("Got payload", zap.Any("payload", payload))

	if payload.UserID == "" {
		return nil, errors.New("internal error: user id is required")
	}

	err := validators.StagingCheckSensitive(ctx, payload.UserID)

	if err != nil {
		return nil, err
	}

	table, idCol, entityLabel, err := entityTarget(payload.ForType)

	if err != nil {
		return nil, err
	}

	switch payload.ProductID {
	case "premium":
		for _, plan := range Plans {
			if plan.ID == payload.ProductName {
				// Ensure the bot/server associated with For exists
				var count int64

				err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE "+idCol+" = $1", payload.For).Scan(&count)

				if err != nil {
					return nil, errors.New("our database broke, please try again later")
				}

				if count == 0 {
					return nil, errors.New(entityLabel + " id is invalid")
				}

				var typeStr string
				var premium bool

				err = state.Pool.QueryRow(ctx, "SELECT type, premium FROM "+table+" WHERE "+idCol+" = $1", payload.For).Scan(&typeStr, &premium)

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

// entityTarget maps a validated for_type to the table/column/label needed
// to build a query against it. table/idCol are only ever one of two
// hardcoded literal pairs (never derived from unvalidated input), so
// string-building the query with them is safe.
func entityTarget(forType string) (table, idCol, label string, err error) {
	switch forType {
	case "bot":
		return "bots", "bot_id", "bot", nil
	case "server":
		return "servers", "server_id", "server", nil
	default:
		return "", "", "", errors.New("invalid for_type")
	}
}

func GivePerks(ctx context.Context, perkData PerkData) error {
	err := validators.StagingCheckSensitive(ctx, perkData.UserID)

	if err != nil {
		return err
	}

	perk, err := FindPerks(ctx, perkData)

	if err != nil {
		return err
	}

	// Check if the user has already purchased this perk, if not give it to them
	switch perkData.ProductID {
	case "premium":
		table, idCol, _, err := entityTarget(perkData.ForType)

		if err != nil {
			return err
		}

		targetID := perkData.For

		_, err = state.Pool.Exec(ctx,
			"UPDATE "+table+" SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE "+idCol+" = $2",
			perk.TimePeriod,
			targetID,
		)

		if err != nil {
			return errors.New("our database broke, please try again later")
		}

		// Servers aren't Discord-mentionable as a user the way a bot's own
		// snowflake is, so the mod log just names it directly.
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

		// The mod log above is staff-only visibility — the purchaser gets
		// nothing confirming their own purchase without this. Best-effort:
		// the premium period is already applied, so a failure here shouldn't
		// turn into an error the caller reads as "the purchase failed".
		if err := notifications.PushNotification(perkData.UserID, types.Alert{
			Type:    types.AlertTypeSuccess,
			Title:   "Premium Activated",
			Message: perkData.For + " (" + perkData.ForType + ") now has premium for the next " + strconv.Itoa(perk.TimePeriod) + " hours.",
		}); err != nil {
			state.Logger.Warn("Failed to send premium confirmation alert", zap.Error(err), zap.String("user_id", perkData.UserID), zap.String("for", targetID))
		}
	}

	return nil
}
