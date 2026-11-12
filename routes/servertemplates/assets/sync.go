// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"encoding/json"
	"errors"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
)

func SourceParams(meta *DiscordTemplateMeta) (pgtype.Text, pgtype.Timestamptz) {
	guild := pgtype.Text{String: meta.SourceGuild, Valid: meta.SourceGuild != ""}
	updated := pgtype.Timestamptz{}

	if meta.UpdatedAt != nil {
		updated = pgtype.Timestamptz{Time: *meta.UpdatedAt, Valid: true}
	}

	return guild, updated
}

func Sync(ctx context.Context, id, code string) (*DiscordTemplateMeta, error) {
	q := db.New(state.Pool)
	meta, err := FetchDiscordTemplate(ctx, code)

	if errors.Is(err, ErrTemplateGone) {
		if markErr := q.MarkServerTemplateUnavailable(ctx, id); markErr != nil {
			return nil, markErr
		}

		return nil, ErrTemplateGone
	}

	if err != nil {
		return nil, err
	}

	channels, err := json.Marshal(meta.Channels)

	if err != nil {
		return nil, err
	}

	roles, err := json.Marshal(meta.Roles)

	if err != nil {
		return nil, err
	}

	guild, updated := SourceParams(meta)

	err = q.SyncServerTemplate(ctx, db.SyncServerTemplateParams{
		ID:              id,
		Name:            meta.Name,
		UsageCount:      meta.UsageCount,
		Channels:        channels,
		Roles:           roles,
		SourceGuildID:   guild,
		SourceUpdatedAt: updated,
		OutOfDate:       meta.OutOfDate,
	})

	if err != nil {
		return nil, err
	}

	return meta, nil
}

func ApplySyncState(t *types.ServerTemplate, icon pgtype.Text, sourceUpdated, synced pgtype.Timestamptz, outOfDate, unavailable bool) {
	if icon.Valid {
		t.IconVersion = &icon.String
	}

	if sourceUpdated.Valid {
		t.SourceUpdatedAt = &sourceUpdated.Time
	}

	if synced.Valid {
		t.SyncedAt = &synced.Time
	}

	t.OutOfDate = outOfDate
	t.Unavailable = unavailable
}
