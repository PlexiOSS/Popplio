package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"popplio/state"

	"github.com/PlexiOSS/Keel/jsonimpl"
)

const discordUserAgent = "Popplio (Omniplex API, +https://omniplex.gg)"

// DiscordTemplateMeta is the subset of Discord's public, unauthenticated
// `GET /guilds/templates/{code}` response this needs -- confirmed working
// with no bot token or auth required.
type DiscordTemplateMeta struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UsageCount  int32  `json:"usage_count"`
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

	var meta DiscordTemplateMeta

	if err := jsonimpl.UnmarshalReader(resp.Body, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse discord's response: %w", err)
	}

	if meta.Name == "" {
		return nil, errors.New("that doesn't look like a valid, existing template code")
	}

	return &meta, nil
}
