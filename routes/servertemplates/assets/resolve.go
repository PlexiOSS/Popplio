package assets

import (
	"context"
	"fmt"

	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/dovewing"
)

func ResolveServerTemplate(ctx context.Context, tmpl *types.ServerTemplate) error {
	owner, err := dovewing.GetUser(ctx, tmpl.OwnerID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying dovewing for owner user: %w", err)
	}

	tmpl.Owner = owner

	if tmpl.Tags == nil {
		tmpl.Tags = []string{}
	}

	// Channels/Roles are only ever populated by the single-template fetch
	// (see GetServerTemplateByID's own doc comment) -- always nil coming
	// out of the list query, which never selects those columns at all.
	if tmpl.Channels == nil {
		tmpl.Channels = []types.TemplateChannel{}
	}

	if tmpl.Roles == nil {
		tmpl.Roles = []types.TemplateRole{}
	}

	return nil
}
