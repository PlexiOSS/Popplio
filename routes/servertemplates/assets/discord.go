package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/jsonimpl"
)

const discordUserAgent = "Popplio (Omniplex API, +https://omniplex.gg)"

// rawDiscordTemplate is Discord's actual `GET /guilds/templates/{code}`
// response shape (the parts this cares about, at least) -- much richer
// than DiscordTemplateMeta below. serialized_source_guild carries a full
// copy of the source guild's channels and roles, which is where the
// channel/role preview comes from; everything else in it (permission
// overwrites, verification level, AFK settings, ...) is discarded.
type rawDiscordTemplate struct {
	Code                  string `json:"code"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	UsageCount            int32  `json:"usage_count"`
	SerializedSourceGuild struct {
		Roles []struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Color int    `json:"color"`
		} `json:"roles"`
		Channels []struct {
			ID       int    `json:"id"`
			Type     int    `json:"type"`
			Name     string `json:"name"`
			Position int    `json:"position"`
			ParentID *int   `json:"parent_id"`
		} `json:"channels"`
	} `json:"serialized_source_guild"`
}

// DiscordTemplateMeta is the subset of Discord's public, unauthenticated
// `GET /guilds/templates/{code}` response this needs -- confirmed working
// with no bot token or auth required.
type DiscordTemplateMeta struct {
	Code        string                  `json:"code"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	UsageCount  int32                   `json:"usage_count"`
	Channels    []types.TemplateChannel `json:"channels"`
	Roles       []types.TemplateRole    `json:"roles"`
}

// FetchDiscordTemplate validates a template code against Discord's own API
// and returns its metadata. A nil, nil-error return never happens -- either
// the template exists (metadata returned) or an error explains why not.
func FetchDiscordTemplate(ctx context.Context, code string) (*DiscordTemplateMeta, error) {
	cli := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, state.Config.Meta.PopplioProxy+"/api/v10/guilds/templates/"+code, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", discordUserAgent)

	resp, err := cli.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to reach discord: %w", err)
	}

	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, errors.New("we're being rate limited by discord, please try again in a moment")
	case resp.StatusCode != http.StatusOK:
		return nil, errors.New("that doesn't look like a valid, existing template code")
	}

	var raw rawDiscordTemplate

	if err := jsonimpl.UnmarshalReader(resp.Body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse discord's response: %w", err)
	}

	if raw.Name == "" {
		return nil, errors.New("that doesn't look like a valid, existing template code")
	}

	channels := make([]types.TemplateChannel, len(raw.SerializedSourceGuild.Channels))
	for i, c := range raw.SerializedSourceGuild.Channels {
		channels[i] = types.TemplateChannel{
			ID:       c.ID,
			Type:     c.Type,
			Name:     c.Name,
			ParentID: c.ParentID,
			Position: c.Position,
		}
	}

	// @everyone is always present and never meaningful to show in a "roles
	// included" preview -- every server has it implicitly.
	roles := make([]types.TemplateRole, 0, len(raw.SerializedSourceGuild.Roles))
	for _, r := range raw.SerializedSourceGuild.Roles {
		if r.Name == "@everyone" {
			continue
		}
		roles = append(roles, types.TemplateRole{ID: r.ID, Name: r.Name, Color: r.Color})
	}

	return &DiscordTemplateMeta{
		Code:        raw.Code,
		Name:        raw.Name,
		Description: raw.Description,
		UsageCount:  raw.UsageCount,
		Channels:    channels,
		Roles:       roles,
	}, nil
}
