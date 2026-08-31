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

	return nil
}
