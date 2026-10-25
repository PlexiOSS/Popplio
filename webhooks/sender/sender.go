// Copyright (C) 2026 NodeByte LTD

package sender

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	rand2 "math/rand"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/events"
	"popplio/webhooks/core/utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/jsonimpl"
)

type webhookData struct {
	ID             string
	Secret         string
	Url            string
	Broken         bool
	FailedRequests int
	SimpleAuth     bool
	HmacAuth       bool
	EventWhitelist []string
}

var (
	ErrNoWebhooks                = errors.New("no webhooks found")
	WebhookMaximumFailedRequests = 20
)

type WebhookEntity struct {
	EntityID   string
	EntityType string
	EntityName string
	SimpleAuth *bool
}

func (e WebhookEntity) Validate() bool {
	return e.EntityID != "" && e.EntityType != "" && e.EntityName != ""
}

type WebhookData struct {
	Event     *events.WebhookResponse
	BadIntent bool
	LogID     string
	UserID    string
	Entity    WebhookEntity
}

type WebhookSendResult struct {
	SendStates map[string]string
}

func Send(d *WebhookData) (*WebhookSendResult, error) {
	if !d.Entity.Validate() {
		return nil, errors.New("invalid webhook entity")
	}

	if d.Event == nil {
		return nil, errors.New("no event set in webhook data")
	}

	q := db.New(state.Pool)

	rawWebhooks, err := q.GetWebhooksForTarget(state.Context, db.GetWebhooksForTargetParams{
		TargetID:   d.Entity.EntityID,
		TargetType: d.Entity.EntityType,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch webhooks: %w", err)
	}

	if len(rawWebhooks) == 0 {
		return nil, ErrNoWebhooks
	}

	webhooks := make([]webhookData, len(rawWebhooks))
	for i, row := range rawWebhooks {
		webhooks[i] = webhookData{
			ID:             row.ID,
			Secret:         row.Secret,
			Url:            row.Url,
			Broken:         row.Broken,
			FailedRequests: int(row.FailedRequests),
			SimpleAuth:     row.SimpleAuth,
			HmacAuth:       row.HmacAuth,
			EventWhitelist: row.EventWhitelist,
		}
	}

	dataBytes, err := jsonimpl.Marshal(d.Event)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var dataMap map[string]any

	if err := jsonimpl.Unmarshal(dataBytes, &dataMap); err != nil {
		return nil, fmt.Errorf("failed to prepare webhook payload for storage: %w", err)
	}

	var webhErrors map[string]error
	var sendStates = make(map[string]string)
	for _, webhook := range webhooks {
		if webhook.Broken || webhook.FailedRequests >= WebhookMaximumFailedRequests {
			continue
		}

		if len(webhook.EventWhitelist) > 0 {
			if !slices.Contains(webhook.EventWhitelist, d.Event.Type) {
				continue
			}
		}

		var logID string
		if d.LogID == "" {
			var webhookUUID pgtype.UUID
			if scanErr := webhookUUID.Scan(webhook.ID); scanErr != nil {
				if webhErrors == nil {
					webhErrors = make(map[string]error)
				}

				webhErrors[webhook.ID] = scanErr

				continue
			}

			logID, err = q.InsertWebhookLogReturningID(state.Context, db.InsertWebhookLogReturningIDParams{
				TargetID:   d.Entity.EntityID,
				TargetType: d.Entity.EntityType,
				UserID:     d.UserID,
				Url:        webhook.Url,
				Data:       dataMap,
				BadIntent:  d.BadIntent,
				WebhookID:  webhookUUID,
			})

			if err != nil {
				if webhErrors == nil {
					webhErrors = make(map[string]error)
				}

				webhErrors[webhook.ID] = err

				continue
			}
		} else {
			logID = d.LogID
		}

		st := &webhookSendState{
			Event:     d.Event,
			BadIntent: d.BadIntent,
			Webhook:   &webhook,
			LogID:     logID,
			UserID:    d.UserID,
			Entity:    d.Entity,
		}

		err = send(st, &webhook, &dataBytes)

		if err != nil {
			if webhErrors == nil {
				webhErrors = make(map[string]error)
			}

			webhErrors[webhook.ID] = err
			sendStates[webhook.ID] = "INTERNAL_ERROR"
			continue
		}

		sendStates[webhook.ID] = st.SendState
	}

	if len(sendStates) == 0 {
		return nil, ErrNoWebhooks
	}

	if len(webhErrors) > 0 {
		var errStr = strings.Builder{}
		for url, err := range webhErrors {
			errStr.WriteString(fmt.Sprintf("%s: %s\n", url, err.Error()))
		}

		return nil, errors.New(errStr.String())
	}

	res := &WebhookSendResult{
		SendStates: sendStates,
	}

	return res, nil
}

func send(d *webhookSendState, webhook *webhookData, pBytes *[]byte) (err error) {
	// Several early-return paths below (a rejected Discord webhook check, a
	// failed SendDiscord call, a failed buildRequest, ...) used to return an
	// error without ever calling d.cancelSend, leaving that attempt's
	// webhook_logs row stuck at its initial PENDING status forever --
	// PullPendingForAll only re-scans pending rows once at process startup
	// (webhooks/setup.go), so nothing ever revisited them. Every other path
	// in this function already calls cancelSend with a specific state before
	// returning; this defer is the backstop that guarantees the same is
	// true of any error return that doesn't, current or future, without
	// having to thread cancelSend into every branch by hand. Explicit
	// cancelSend calls elsewhere in this function still win (cancelSend's
	// own "first call wins" guard, and the d.SendState == "" check here,
	// both prevent this from overwriting a more specific state).
	defer func() {
		if err != nil && d.SendState == "" {
			d.cancelSend("INTERNAL_ERROR")
		}
	}()

	if !d.Entity.Validate() {
		return errors.New("invalid webhook entity")
	}

	if pBytes == nil {
		return errors.New("pBytes is nil")
	}

	data := *pBytes

	if !d.BadIntent {
		prefix, err := utils.GetDiscordWebhookInfo(webhook.Url)

		if err != nil && !errors.Is(err, utils.ErrNotActuallyWebhook) {
			return fmt.Errorf("error while checking webhook: %w", err)
		}

		if prefix != "" && !errors.Is(err, utils.ErrNotActuallyWebhook) {
			params := d.Event.Data.CreateDiscordEmbed(d.Event.Creator, d.Event.Targets)

			err = SendDiscord(
				webhook.Url,
				prefix,
				params,
			)

			if errors.Is(err, ErrDiscordWebhookGone) {
				if markErr := d.markBroken(); markErr != nil {
					state.Logger.Error("Failed to mark Discord webhook broken", d.logFields(zap.Error(markErr))...)
				}
			}

			if err != nil {
				return fmt.Errorf("failed to send discord webhook: %w", err)
			}

			d.markSucceeded()
			d.cancelSend("SUCCESS")
			return nil
		}
	}

	if err := d.resolveTarget(webhook.Url); err != nil {
		return err
	}

	if !d.BadIntent {
		if rand2.Float64() < 0.4 {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						state.Logger.Error("Panic in bad-intent webhook test-send", d.logFields(zap.Any("panic", r))...)
					}
				}()

				var dataMap map[string]any
				if err := jsonimpl.Unmarshal(data, &dataMap); err != nil {
					state.Logger.Error("Failed to prepare bad-intent webhook payload for storage", d.logFields(zap.Error(err))...)
					return
				}

				var webhookUUID pgtype.UUID
				if err := webhookUUID.Scan(webhook.ID); err != nil {
					state.Logger.Error("Failed to parse webhook id for bad-intent probe", d.logFields(zap.Error(err))...)
					return
				}

				logID, err := db.New(state.Pool).InsertWebhookLogReturningID(state.Context, db.InsertWebhookLogReturningIDParams{
					TargetID:   d.Entity.EntityID,
					TargetType: d.Entity.EntityType,
					UserID:     d.UserID,
					Url:        webhook.Url,
					Data:       dataMap,
					BadIntent:  true,
					WebhookID:  webhookUUID,
				})

				if err != nil {
					state.Logger.Error("Failed to insert webhook log", d.logFields(zap.Error(err))...)
					return
				}

				badD := &webhookSendState{
					Event:       d.Event,
					BadIntent:   true,
					Webhook:     webhook,
					UserID:      d.UserID,
					LogID:       logID,
					Entity:      d.Entity,
					ResolvedIps: d.ResolvedIps,
				}

				send(badD, webhook, pBytes)
			}()
		}
	}

	state.Logger.Info("Sending webhook", d.logFields()...)

	req, err := d.buildRequest(webhook, data)

	if err != nil {
		return err
	}

	resp, err := webhookClient.Do(req)

	if err != nil {
		state.Logger.Error("Failed to send webhook", d.logFields(zap.Error(err))...)
		d.cancelSend("REQUEST_SEND_FAILURE")
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if err != nil {
		body = []byte("Failed to read body: " + err.Error())
	}

	var reqHeaders = map[string]string{}
	for k, v := range req.Header {
		if len(reqHeaders) > 10 {
			break
		}

		reqHeaders[k] = strings.Join(v, ",")
	}

	var respHeaders = map[string]string{}
	for k, v := range resp.Header {
		if len(respHeaders) > 20 {
			break
		}

		respHeaders[k] = strings.Join(v, ",")
	}

	reqHeadersAny := make(map[string]any, len(reqHeaders))
	for k, v := range reqHeaders {
		reqHeadersAny[k] = v
	}

	respHeadersAny := make(map[string]any, len(respHeaders))
	for k, v := range respHeaders {
		respHeadersAny[k] = v
	}

	var logUUID pgtype.UUID
	if err := logUUID.Scan(d.LogID); err != nil {
		state.Logger.Error("Failed to parse webhook log id", d.logFields(zap.Error(err))...)
		return fmt.Errorf("failed to parse webhook log id: %w", err)
	}

	err = db.New(state.Pool).UpdateWebhookLogResponse(state.Context, db.UpdateWebhookLogResponseParams{
		Response:        pgtype.Text{String: string(body), Valid: true},
		StatusCode:      int32(resp.StatusCode),
		RequestHeaders:  reqHeadersAny,
		ResponseHeaders: respHeadersAny,
		ID:              logUUID,
	})

	if err != nil {
		state.Logger.Error("Failed to update webhook logs with response", d.logFields(zap.Error(err))...)
		return fmt.Errorf("failed to update webhook logs with response: %w", err)
	}

	switch {
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		d.cancelSend("WEBHOOK_404_410")

		if err := d.markBroken(); err != nil {
			state.Logger.Error("Failed to mark webhook broken", d.logFields(zap.Error(err))...)
			return fmt.Errorf("webhook returned %d and could not be marked broken: %w", resp.StatusCode, err)
		}

		d.notify(types.AlertTypeWarning, "Webhook disabled", fmt.Sprintf("A webhook for %s returned %d, so it's been disabled. Fix the URL and save the webhook again to turn it back on.", d.Entity.EntityName, resp.StatusCode))

		return fmt.Errorf("webhook returned %d and was marked broken", resp.StatusCode)

	case resp.StatusCode == 401 || resp.StatusCode == 403:
		if d.BadIntent {
			d.cancelSend("SUCCESS")

			return nil
		} else {
			d.cancelSend("WEBHOOK_AUTH_INVALID")

			d.notify(types.AlertTypeError, "Webhook Auth Error", "Webhook could not be securely authenticated by the bot at this time. Please try again later.")

			if err := d.markFailed(); err != nil {
				return errors.New("webhook failed to validate auth and failed to mark request as failed")
			}

			return errors.New("webhook auth error:" + strconv.Itoa(resp.StatusCode))
		}

	case resp.StatusCode > 400:
		d.cancelSend("RESPONSE_" + strconv.Itoa(resp.StatusCode))

		d.notify(types.AlertTypeError, "Webhook Auth Error", fmt.Sprintf("We were unable to notify this bot: %d", resp.StatusCode))

		return errors.New("webhook returned error: " + strconv.Itoa(resp.StatusCode))

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if d.BadIntent {
			d.cancelSend("WEBHOOK_BROKEN_BAD_AUTHCODE")

			d.notify(types.AlertTypeWarning, "Webhook isn't verifying signatures", fmt.Sprintf("A webhook for %s accepted a request signed with the wrong secret. Anyone could send it fake events. Check the signature on every request and reject invalid ones with a 401 or 403.", d.Entity.EntityName))

			return errors.New("webhook accepted a request with an invalid signature")
		}

		d.markSucceeded()
		d.cancelSend("SUCCESS")

		d.notify(types.AlertTypeSuccess, "Webhook Send Successful!", "Successfully notified "+d.Entity.EntityName+" of this action.")
	}

	return nil
}

var ErrDiscordWebhookGone = errors.New("discord webhook no longer exists or is unauthorized")

func SendDiscord(url, prefix string, params *discord.Embed) error {
	url = state.Config.Meta.PopplioProxy + "/" + strings.TrimPrefix(url, prefix)

	payload, err := jsonimpl.Marshal(discord.WebhookMessageCreate{
		Embeds: []discord.Embed{*params},
	})

	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 401, 403, 404, 410:
		return fmt.Errorf("%w: %d", ErrDiscordWebhookGone, resp.StatusCode)
	}

	return nil
}
