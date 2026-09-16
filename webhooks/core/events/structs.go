// Copyright (C) 2026 NodeByte LTD

package events

import (
	"fmt"
	"time"

	"popplio/types"

	"github.com/PlexiOSS/Keel/ptr"

	"github.com/disgoorg/disgo/discord"
	"github.com/mitchellh/mapstructure"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/PlexiOSS/Keel/jsonimpl"
)

type Target struct {
	Bot    *dovetypes.PlatformUser `json:"bot,omitempty" description:"If a bot event, the bot that the webhook is about"`
	Server *types.IndexServer      `json:"server,omitempty" description:"If a server event, the server that the webhook is about"`
	Team   *types.Team             `json:"team,omitempty" description:"If a team event, the team that the webhook is about"`
}

type WebhookResponse struct {
	Creator  *dovetypes.PlatformUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	Type     string                  `json:"type" dynexample:"true" description:"The type of the webhook event"`
	Data     WebhookEvent            `json:"data" dynschema:"true" description:"The data of the webhook event"`
	Targets  Target                  `json:"targets" description:"The target of the webhook, can be one of. or a possible combination of bot, team and server"`
	Metadata WebhookMetadata         `json:"metadata" description:"Metadata about the webhook event"`
}

func (wr *WebhookResponse) UnmarshalJSON(b []byte) error {
	var smap map[string]any

	err := jsonimpl.Unmarshal(b, &smap)

	if err != nil {
		return fmt.Errorf("failed to unmarshal webhook response: %w", err)
	}

	typ, ok := smap["type"].(string)

	if !ok {
		return fmt.Errorf("failed to unmarshal webhook response: type not a string")
	}

	evt, ok := eventMapToType[typ]

	if !ok {
		return fmt.Errorf("failed to unmarshal webhook response: invalid type")
	}

	wr.Type = typ
	wr.Data = evt

	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   wr,
		TagName:  "json",
		Squash:   true,
	}

	decoder, e := mapstructure.NewDecoder(cfg)

	if e != nil {
		return e
	}

	e = decoder.Decode(smap)

	if e != nil {
		return e
	}

	return nil
}

type Changeset[T any] struct {
	Old T `json:"old"`
	New T `json:"new"`
}

type WebhookMetadata struct {
	CreatedAt int64 `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Test      bool  `json:"test" description:"Whether the vote was a test vote or not"`
}

func ParseWebhookMetadata(w *WebhookMetadata) WebhookMetadata {
	if w == nil {
		w = &WebhookMetadata{}
	}

	if w.CreatedAt == 0 {
		w.CreatedAt = time.Now().Unix()
	}

	return *w
}

func ConvertChangesetToEmbedFields[T any](name string, c Changeset[T]) []discord.EmbedField {
	return []discord.EmbedField{
		{
			Name: "Old " + name,
			Value: func() string {
				if len(fmt.Sprint(c.Old)) > 1000 {
					return fmt.Sprint(c.Old)[:1000] + "..."
				}

				return fmt.Sprint(c.Old)
			}(),
			Inline: ptr.TruePtr,
		},
		{
			Name: "New " + name,
			Value: func() string {
				if len(fmt.Sprint(c.New)) > 1000 {
					return fmt.Sprint(c.New)[:1000] + "..."
				}

				return fmt.Sprint(c.New)
			}(),
			Inline: ptr.TruePtr,
		},
	}
}

func (t Target) GetBestTargetType() string {
	if t.Bot != nil {
		return "bot"
	}

	if t.Server != nil {
		return "server"
	}

	if t.Team != nil {
		return "team"
	}

	return "<unknown>"
}

func (t Target) GetTargetTypes() []string {
	var types []string

	if t.Bot != nil {
		types = append(types, "bot")
	}

	if t.Server != nil {
		types = append(types, "server")
	}

	if t.Team != nil {
		types = append(types, "team")
	}

	return types
}

func (t Target) GetID() string {
	if t.Bot != nil {
		return t.Bot.ID
	}

	if t.Server != nil {
		return t.Server.ServerID
	}

	if t.Team != nil {
		return t.Team.ID
	}

	return "<unknown>"
}

func (t Target) GetUsername() string {
	if t.Bot != nil {
		return t.Bot.Username
	}

	if t.Server != nil {
		return t.Server.Name
	}

	if t.Team != nil {
		return t.Team.Name
	}

	return "<unknown>"
}

func (t Target) GetDisplayName() string {
	if t.Bot != nil {
		return t.Bot.DisplayName
	}

	if t.Server != nil {
		return t.Server.Name
	}

	if t.Team != nil {
		return t.Team.Name
	}

	return "<unknown>"
}

func (t Target) GetAvatarURL() string {
	if t.Bot != nil {
		return t.Bot.Avatar
	}

	return ""
}

func (t Target) GetTargetName() string {
	return t.GetBestTargetType() + " " + t.GetUsername()
}

func (t Target) GetURL() string {
	var category string

	switch {
	case t.Bot != nil:
		category = "bots/"
	case t.Server != nil:
		category = "servers/"
	case t.Team != nil:
		category = "teams/"
	}

	return "https://omniplex.gg/" + category + t.GetID()
}

func (t Target) GetTargetLink(header, path string) string {
	return "[" + header + " " + t.GetUsername() + "](" + t.GetURL() + path + ")"
}

func (t Target) GetViewLink() string {
	return t.GetTargetLink("View", "")
}
