package notifications

import (
	"errors"
	"fmt"

	"popplio/state"
	"popplio/types"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/jsonimpl"
)

func PushNotification(userId string, notif types.Alert) error {
	err := state.Validator.Struct(notif)

	if err != nil {
		return fmt.Errorf("invalid notification: %s", err)
	}

	if len(notif.AlertData) == 0 {
		notif.AlertData = map[string]any{}
	}

	enabled, err := categoryEnabled(userId, notif.Category)

	if err != nil {
		return fmt.Errorf("failed to check notification preference: %s", err)
	}

	if !enabled {
		return nil
	}

	// NoSave means what it says: true skips persisting the alert to the
	// inbox, false (the zero value, so most callers get this by default)
	// saves it. Muting an entire category via user_notification_prefs
	// (checked above) is the normal way to quiet a noisy alert type now --
	// NoSave is only for a genuinely one-off, ephemeral push that was never
	// meant to be readable later. This used to be inverted (`if
	// notif.NoSave`), which meant every "normal" alert silently never
	// reached a user's inbox — only the one caller that explicitly opted
	// OUT of saving ever actually persisted anything.
	if !notif.NoSave {
		_, err = state.Pool.Exec(
			state.Context,
			"INSERT INTO alerts (user_id, type, url, message, title, icon, alert_data, priority, category) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			userId,
			notif.Type,
			notif.URL,
			notif.Message,
			notif.Title,
			notif.Icon,
			notif.AlertData,
			notif.Priority,
			notif.Category,
		)

		if err != nil {
			state.Logger.Error("Error inserting alert", zap.Error(err), zap.String("user_id", userId), zap.Any("alert", notif))
			return err
		}
	}

	bytes, err := jsonimpl.Marshal(notif)

	if err != nil {
		return err
	}

	notifIds, err := state.Pool.Query(state.Context, "SELECT notif_id, auth, endpoint, p256dh FROM user_notifications WHERE user_id = $1", userId)

	if err != nil {
		return err
	}

	defer notifIds.Close()

	for notifIds.Next() {
		var notifId string
		var auth string
		var endpoint string
		var p256dh string

		err = notifIds.Scan(&notifId, &auth, &endpoint, &p256dh)

		if err != nil {
			return fmt.Errorf("error finding notification: %s", err)
		}

		if notifId == "" {
			continue
		}

		state.Logger.Info("Sending notification", zap.String("notif_id", notifId), zap.String("endpoint", endpoint))

		sub := webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				Auth:   auth,
				P256dh: p256dh,
			},
		}

		resp, err := webpush.SendNotification(bytes, &sub, &webpush.Options{
			Subscriber:      "notifications@infinitybots.gg",
			VAPIDPublicKey:  state.Config.Notifications.VapidPublicKey,
			VAPIDPrivateKey: state.Config.Notifications.VapidPrivateKey,
			TTL:             30,
		})

		if err != nil {
			if resp.StatusCode == 410 || resp.StatusCode == 404 {
				state.Pool.Exec(state.Context, "DELETE FROM user_notifications WHERE notif_id = $1", notifId)
			}
			return err
		}
	}

	return nil
}

// categoryEnabled reports whether userId wants notifications for category.
// No row in user_notification_prefs means enabled -- this is an opt-out
// model, so existing users keep seeing everything until they explicitly
// mute a category, no backfill required.
func categoryEnabled(userId string, category types.AlertCategory) (bool, error) {
	var enabled bool

	err := state.Pool.QueryRow(
		state.Context,
		"SELECT enabled FROM user_notification_prefs WHERE user_id = $1 AND category = $2",
		userId, category,
	).Scan(&enabled)

	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return enabled, nil
}
