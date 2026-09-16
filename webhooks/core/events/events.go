// Copyright (C) 2026 NodeByte LTD

package events

import (
	"reflect"

	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

type EventRegistry struct {
	Event    WebhookEvent
	TestVars []types.TestWebhookVariables
}

var Registry = []EventRegistry{}

type WebhookEvent interface {
	TargetTypes() []string
	Event() string
	CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets Target) *discord.Embed
	Summary() string
	Description() string
}

var eventList = []WebhookEvent{}
var eventMapToType = map[string]WebhookEvent{}

func AddEvent(a WebhookEvent) {
	eventList = append(eventList, a)
}

func RegisterAddedEvents() {
	for _, a := range eventList {
		state.Logger.Error("Webhook event register", zap.String("event", a.Event()))
		registerEventImpl(a)
	}
}

func registerEventImpl(a WebhookEvent) {
	eventMapToType[a.Event()] = a

	docs.AddWebhook(&docs.WebhookDoc{
		Name:    a.Event(),
		Summary: a.Summary(),
		Tags: []string{
			"Webhooks",
		},
		Description: a.Description(),
		Format: WebhookResponse{
			Type: a.Event(),
			Data: a,
		},
		FormatName: "WEBHOOK-" + a.Event(),
	})

	changesetOf := func(t types.WebhookType) types.WebhookType {
		return types.WebhookType(string(types.WebhookTypeChangeset) + "/" + string(t))
	}

	evt := EventRegistry{
		Event: a,
	}

	refType := reflect.TypeOf(a)

	var cols []types.TestWebhookVariables

	for _, f := range reflect.VisibleFields(refType) {
		var fieldType string

		switch f.Type.Kind() {
		case reflect.String:
			fieldType = types.WebhookTypeText
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = types.WebhookTypeNumber
		case reflect.Bool:
			fieldType = types.WebhookTypeBoolean
		case reflect.Struct:
			ti := reflect.Zero(f.Type).Interface()

			switch ti.(type) {
			case Changeset[[]string]:
				fieldType = changesetOf(types.WebhookTypeTextArray)
			case Changeset[[]types.Link]:
				fieldType = changesetOf(types.WebhookTypeLinkArray)
			case Changeset[string]:
				fieldType = changesetOf(types.WebhookTypeText)
			case Changeset[int], Changeset[int8], Changeset[int16], Changeset[int32], Changeset[int64]:
				fieldType = changesetOf(types.WebhookTypeNumber)
			case Changeset[bool]:
				fieldType = changesetOf(types.WebhookTypeBoolean)
			default:
				panic("Illegal field type: " + string(a.Event()) + "->" + f.Name + " <struct>")
			}
		default:
			panic("Illegal field type: " + string(a.Event()) + "->" + f.Name)
		}

		if f.Tag.Get("json") == "" {
			panic("Json tag missing: " + string(a.Event()) + "->" + f.Name)
		}

		var label = f.Name

		if f.Tag.Get("testlabel") != "" {
			label = f.Tag.Get("testlabel")
		}

		cols = append(cols, types.TestWebhookVariables{
			ID:          f.Tag.Get("json"),
			Name:        label,
			Description: f.Tag.Get("description"),
			Value:       f.Tag.Get("testvalue"),
			Type:        fieldType,
		})
	}

	evt.TestVars = cols

	Registry = append(Registry, evt)
}
