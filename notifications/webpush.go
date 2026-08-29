package notifications

import (
	"errors"
	"fmt"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

	q := db.New(state.Pool)

	if !notif.NoSave {
		err = q.InsertAlert(state.Context, db.InsertAlertParams{
			UserID:    userId,
			Type:      notif.Type,
			Url:       notif.URL,
			Message:   notif.Message,
			Title:     notif.Title,
			Icon:      pgtype.Text{String: notif.Icon, Valid: notif.Icon != ""},
			AlertData: notif.AlertData,
			Priority:  notif.Priority,
			Category:  notif.Category,
		})

		if err != nil {
			state.Logger.Error("Error inserting alert", zap.Error(err), zap.String("user_id", userId), zap.Any("alert", notif))
			return err
		}
	}

	bytes, err := jsonimpl.Marshal(notif)

	if err != nil {
		return err
	}

	subs, err := q.GetUserNotificationSubscriptions(state.Context, userId)

	if err != nil {
		return err
	}

	for _, row := range subs {
		notifId, auth, endpoint, p256dh := row.NotifID, row.Auth, row.Endpoint, row.P256dh

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
			Subscriber:      "notifications@omniplex.gg",
			VAPIDPublicKey:  state.Config.Notifications.VapidPublicKey,
			VAPIDPrivateKey: state.Config.Notifications.VapidPrivateKey,
			TTL:             30,
		})

		if err != nil {
			if resp.StatusCode == 410 || resp.StatusCode == 404 {
				q.DeleteUserNotificationByNotifID(state.Context, notifId)
			}
			return err
		}
	}

	return nil
}

func categoryEnabled(userId string, category types.AlertCategory) (bool, error) {
	enabled, err := db.New(state.Pool).GetUserNotificationPrefEnabled(state.Context, db.GetUserNotificationPrefEnabledParams{
		UserID:   userId,
		Category: category,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return enabled, nil
}
