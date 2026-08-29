package drivers

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"
	"popplio/webhooks/sender"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/dovewing"
)

type Driver interface {
	Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error)
	TargetType() string
	CanBeConstructed(userId, targetId string) (bool, error)
	SupportsPullPending(userId, targetId string) (bool, error)
}

var DriverRegistry = map[string]Driver{}

func RegisterDriver(driver Driver) {
	DriverRegistry[driver.TargetType()] = driver
}

type With struct {
	UserID     string
	TargetID   string
	TargetType string
	Metadata   *events.WebhookMetadata
	Data       events.WebhookEvent
}

func Send(with With) error {
	targetTypes := with.Data.TargetTypes()
	if !slices.Contains(targetTypes, with.TargetType) {
		return errors.New("invalid event type")
	}

	driver, ok := DriverRegistry[with.TargetType]

	if !ok {
		return errors.New("target type not registered")
	}

	supports, err := driver.CanBeConstructed(with.UserID, with.TargetID)

	if err != nil {
		return fmt.Errorf("failed to check if entity supports construction: %w", err)
	}

	if !supports {
		return nil
	}

	target, entity, err := driver.Construct(with.UserID, with.TargetID)

	if err != nil {
		return err
	}

	if entity == nil {
		return errors.New("failed to construct webhook entity due to no entity being returned")
	}

	if entity.EntityType != with.TargetType {
		return fmt.Errorf("entity type mismatch: expected %s, got %s", with.TargetType, entity.EntityType)
	}

	user, err := dovewing.GetUser(state.Context, with.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to fetch user via dovewing for this hook", zap.Error(err), zap.String("targetType", with.TargetType), zap.String("targetID", with.TargetID), zap.String("userID", with.UserID))
		return err
	}

	resp := &events.WebhookResponse{
		Creator:  user,
		Targets:  *target,
		Type:     with.Data.Event(),
		Data:     with.Data,
		Metadata: events.ParseWebhookMetadata(with.Metadata),
	}

	d := &sender.WebhookData{
		UserID: resp.Creator.ID,
		Entity: *entity,
		Event:  resp,
	}

	res, err := sender.Send(d)

	if err != nil && !errors.Is(err, sender.ErrNoWebhooks) {
		perr := notifications.PushNotification(d.UserID, types.Alert{
			Type:     types.AlertTypeError,
			Message:  fmt.Sprintf("Failed to send webhooks: %s", err.Error()),
			Title:    "Webhook Send Failed",
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/dashboard", Valid: true},
			Category: types.AlertCategoryWebhooks,
		})

		if perr != nil {
			state.Logger.Error("Error when push notification for erroring webhook", zap.Error(err), zap.String("logID", d.LogID), zap.String("userID", d.UserID), zap.String("entityID", d.Entity.EntityID), zap.Any("sendState", res.SendStates))
		}
	}

	return err
}

func PullPending(p Driver) error {
	targetType := p.TargetType()

	q := db.New(state.Pool)

	rows, err := q.GetPendingWebhookLogsForPull(state.Context, db.GetPendingWebhookLogsForPullParams{
		State:      "PENDING",
		TargetType: targetType,
	})

	if err != nil {
		return fmt.Errorf("failed to fetch pending webhooks: %w", err)
	}

	var eventData []struct {
		ID       string
		TargetID string
		UserID   string
		Event    *events.WebhookResponse
	}

	for _, row := range rows {
		dataBytes, err := json.Marshal(row.Data)

		if err != nil {
			state.Logger.Error("Failed to re-marshal pending webhook data", zap.Error(err))
			continue
		}

		var event events.WebhookResponse

		if err := json.Unmarshal(dataBytes, &event); err != nil {
			state.Logger.Error("Failed to unmarshal pending webhook event", zap.Error(err))
			continue
		}

		eventData = append(eventData, struct {
			ID       string
			TargetID string
			UserID   string
			Event    *events.WebhookResponse
		}{ID: row.ID, TargetID: row.TargetID, UserID: row.UserID, Event: &event})
	}

	for _, v := range eventData {
		state.Logger.Info("Pulled event", zap.Any("event", v.Event), zap.Bool("isTestEvent", v.Event.Metadata.Test))

		supports, err := p.SupportsPullPending(v.UserID, v.TargetID)

		if err != nil {
			state.Logger.Error("Failed to check if entity supports pulls", zap.Error(err), zap.String("targetId", v.TargetID), zap.String("targetType", targetType))
			return fmt.Errorf("failed to check if entity supports pulls: %w", err)
		}

		if !supports {
			continue
		}

		_, entity, err := p.Construct(v.UserID, v.TargetID)

		if err != nil {
			state.Logger.Error("Failed to get entity for webhook", zap.Error(err), zap.String("entityID", v.TargetID))
			continue
		}

		if entity.EntityType != targetType {
			return fmt.Errorf("entity type mismatch: expected %s, got %s", targetType, entity.EntityType)
		}

		_, err = sender.Send(&sender.WebhookData{
			Event:  v.Event,
			LogID:  v.ID,
			UserID: v.UserID,
			Entity: *entity,
		})

		if errors.Is(err, sender.ErrNoWebhooks) {
			var logUUID pgtype.UUID
			if scanErr := logUUID.Scan(v.ID); scanErr != nil {
				state.Logger.Error("Failed to parse webhook log id", zap.Error(scanErr), zap.String("entityID", v.TargetID))
				continue
			}

			err = q.MarkWebhookLogNoWebhooks(state.Context, db.MarkWebhookLogNoWebhooksParams{
				State: "NO_WEBHOOKS",
				ID:    logUUID,
			})

			if err != nil {
				state.Logger.Error("Failed to update webhook state", zap.Error(err), zap.String("entityID", v.TargetID))
				continue
			}
		}

		if err != nil {
			state.Logger.Error("Failed to send pending webhook", zap.Error(err), zap.String("entityID", v.TargetID))
			continue
		}
	}

	return nil
}

func PullPendingForAll() error {
	for _, v := range DriverRegistry {
		err := PullPending(v)

		if err != nil {
			return err
		}
	}

	return nil
}
