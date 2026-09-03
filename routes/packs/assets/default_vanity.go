// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"popplio/db"

	"github.com/jackc/pgx/v5"
)

// defaultSlug turns an emoji/sticker's own name into a clean kebab-case
// vanity code: lowercased, spaces/dashes/underscores collapsed to single
// dashes, and anything else (punctuation, emoji glyphs, non-ASCII) dropped
// outright rather than just stripped like patch_vanity's owner-typed-input
// transform does -- an auto-generated default should always come out clean.
func defaultSlug(name string) string {
	var b strings.Builder
	lastDash := false

	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}

	return strings.TrimSuffix(b.String(), "-")
}

// EnsureDefaultVanity gives a pack emoji/sticker a vanity code the moment
// it's created, the same way bots/servers/teams never sit at "no vanity" --
// only pack emojis/stickers were left showing "Not set" until an owner
// manually configured one, which this closes.
//
// A no-op if targetID already has a vanity row -- this is what makes it
// safe to call from patch_pack, which deletes and reinserts a pack's whole
// emoji/sticker list on every save: an item that already had a vanity
// (default or owner-customized) keeps it, since its ID is preserved across
// the delete+reinsert; only a genuinely new ID gets a fresh default here.
func EnsureDefaultVanity(ctx context.Context, q *db.Queries, targetID, targetType, name string) error {
	count, err := q.CountVanityByTarget(ctx, db.CountVanityByTargetParams{
		TargetID:   targetID,
		TargetType: targetType,
	})

	if err != nil {
		return fmt.Errorf("checking for an existing vanity: %w", err)
	}

	if count > 0 {
		return nil
	}

	slug := defaultSlug(name)

	if slug == "" {
		slug = strings.ReplaceAll(targetID, "-", "")[:8]
	}

	codeCount, err := q.CountVanityByCode(ctx, slug)

	if err != nil {
		return fmt.Errorf("checking for a vanity code collision: %w", err)
	}

	if codeCount > 0 {
		suffix := strings.ReplaceAll(targetID, "-", "")
		if len(suffix) > 6 {
			suffix = suffix[:6]
		}
		slug = slug + "-" + suffix
	}

	if err := q.InsertVanity(ctx, db.InsertVanityParams{
		TargetID:   targetID,
		TargetType: targetType,
		Code:       slug,
	}); err != nil {
		return fmt.Errorf("inserting default vanity: %w", err)
	}

	return nil
}

// ResolveVanityCode returns a target's current vanity code, or "" if it has
// none. Errors other than "not found" are returned so callers can decide
// how to handle a real lookup failure rather than silently showing no
// vanity.
func ResolveVanityCode(ctx context.Context, q *db.Queries, targetID string) (string, error) {
	v, err := q.ResolveVanityByTargetID(ctx, targetID)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return v.Code, nil
}
