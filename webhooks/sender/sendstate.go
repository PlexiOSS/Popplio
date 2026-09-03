// Copyright (C) 2026 NodeByte LTD

package sender

import (
	"popplio/db"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type webhookSendState struct {
	Event       *events.WebhookResponse
	BadIntent   bool
	Webhook     *webhookData
	LogID       string
	UserID      string
	Entity      WebhookEntity
	SendState   string
	ResolvedIps []string
}

func (st *webhookSendState) logFields(extra ...zap.Field) []zap.Field {
	return append([]zap.Field{
		zap.String("logID", st.LogID),
		zap.String("userID", st.UserID),
		zap.String("entityID", st.Entity.EntityID),
		zap.Bool("badIntent", st.BadIntent),
	}, extra...)
}

func (st *webhookSendState) cancelSend(saveState string) {
	if saveState != "SUCCESS" {
		state.Logger.Info("Cancelling webhook send", st.logFields()...)
	}

	if st.SendState != "" {
		state.Logger.Warn("SendState is already set", st.logFields(zap.String("sendState", st.SendState))...)
		return
	}

	st.SendState = saveState

	if st.LogID != "" {
		var logUUID pgtype.UUID
		if err := logUUID.Scan(st.LogID); err != nil {
			state.Logger.Error("Failed to parse webhook log id", st.logFields(zap.Error(err))...)
			return
		}

		err := db.New(state.Pool).IncrementWebhookLogTries(state.Context, db.IncrementWebhookLogTriesParams{
			State: saveState,
			ID:    logUUID,
		})

		if err != nil {
			state.Logger.Error("Failed to update webhook logs with new status", st.logFields(zap.Error(err))...)
		}
	}
}

func (st *webhookSendState) notify(alertType types.AlertType, title, message string) {
	err := notifications.PushNotification(st.UserID, types.Alert{
		Type:     alertType,
		Message:  message,
		Title:    title,
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/dashboard", Valid: true},
		Category: types.AlertCategoryWebhooks,
	})

	if err != nil {
		state.Logger.Error("Failed to send notification", st.logFields(zap.Error(err))...)
	}
}

func (st *webhookSendState) markFailed() error {
	return db.New(state.Pool).IncrementWebhookFailedRequests(state.Context, db.IncrementWebhookFailedRequestsParams{
		TargetID:   st.Entity.EntityID,
		TargetType: st.Entity.EntityType,
	})
}
