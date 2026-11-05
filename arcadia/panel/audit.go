// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"popplio/audit"
	"popplio/db"
	"popplio/state"
)

var readOnlyQueries = map[string]bool{
	"Hello":            true,
	"BaseAnalytics":    true,
	"GetUser":          true,
	"BotQueue":         true,
	"ServerQueue":      true,
	"GetRpcMethods":    true,
	"GetRpcLogEntries": true,
	"SearchEntitys":    true,
}

var secretFields = map[string]bool{
	"login_token": true,
	"token":       true,
	"code":        true,
	"otp":         true,
	"secret":      true,
	"password":    true,
}

func variantOf(raw json.RawMessage) (string, json.RawMessage) {
	var name string

	if json.Unmarshal(raw, &name) == nil {
		return name, nil
	}

	var obj map[string]json.RawMessage

	if json.Unmarshal(raw, &obj) == nil && len(obj) == 1 {
		for k, v := range obj {
			return k, v
		}
	}

	return "", nil
}

func isRead(variant string) bool {
	return strings.HasPrefix(variant, "List") || strings.HasPrefix(variant, "Get")
}

func scrub(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))

		for k, val := range x {
			if secretFields[k] {
				continue
			}

			out[k] = scrub(val)
		}

		return out
	case []any:
		for i := range x {
			x[i] = scrub(x[i])
		}

		return x
	default:
		return v
	}
}

func findString(v any, keys ...string) string {
	switch x := v.(type) {
	case map[string]any:
		for _, k := range keys {
			if s, ok := x[k].(string); ok && s != "" {
				return s
			}
		}

		for _, val := range x {
			if s := findString(val, keys...); s != "" {
				return s
			}
		}
	case []any:
		for _, val := range x {
			if s := findString(val, keys...); s != "" {
				return s
			}
		}
	}

	return ""
}

func panelEvent(body []byte) (audit.Event, string, bool) {
	var top map[string]json.RawMessage

	if json.Unmarshal(body, &top) != nil || len(top) != 1 {
		return audit.Event{}, "", false
	}

	var name string
	var raw json.RawMessage

	for k, v := range top {
		name, raw = k, v
	}

	if readOnlyQueries[name] {
		return audit.Event{}, "", false
	}

	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)

	var token string
	_ = json.Unmarshal(fields["login_token"], &token)

	action := "panel." + name

	if name == "Authorize" {
		if variant, _ := variantOf(fields["action"]); variant != "" {
			action += "." + variant
		}

		return audit.Event{Source: audit.SourcePanel, ActorType: audit.ActorStaff, Action: action}, "", true
	}

	if name == "PopplioStaff" {
		var proxy struct {
			Path   string `json:"path"`
			Method string `json:"method"`
		}

		_ = json.Unmarshal(raw, &proxy)

		if strings.EqualFold(proxy.Method, http.MethodGet) {
			return audit.Event{}, "", false
		}
	}

	if actionRaw, ok := fields["action"]; ok {
		variant, _ := variantOf(actionRaw)

		if variant == "" || isRead(variant) {
			return audit.Event{}, "", false
		}

		action += "." + variant
	}

	if name == "ExecuteRpc" {
		if variant, _ := variantOf(fields["method"]); variant != "" {
			action += "." + variant
		}
	}

	var payload any
	_ = json.Unmarshal(raw, &payload)
	payload = scrub(payload)

	data, _ := payload.(map[string]any)

	targetType, _ := data["target_type"].(string)

	return audit.Event{
		Source:     audit.SourcePanel,
		ActorType:  audit.ActorStaff,
		Action:     action,
		TargetType: targetType,
		TargetID:   findString(payload, "target_id", "bot_id", "server_id", "user_id", "id", "itag"),
		Data:       data,
	}, token, true
}

func recordPanel(ctx context.Context, body []byte, status int) {
	event, token, ok := panelEvent(body)

	if !ok {
		return
	}

	if token != "" {
		if chain, err := db.New(state.Pool).GetAuthChainByToken(ctx, token); err == nil {
			event.ActorID = chain.UserID
		}
	}

	event.Status = status
	event.IP = clientIPFromContext(ctx)

	audit.Record(event)
}
