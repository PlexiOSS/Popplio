// Package assets resolves vanity URLs to the entity they point at.
package assets

import (
	"context"
	"errors"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func toVanity(v db.Vanity) *types.Vanity {
	return &types.Vanity{
		ITag:       v.Itag,
		TargetID:   v.TargetID,
		TargetType: v.TargetType,
		Code:       v.Code,
		CreatedAt:  v.CreatedAt.Time,
	}
}

func resolveByTargetID(ctx context.Context, targetID string) (*types.Vanity, error) {
	v, err := db.New(state.Pool).ResolveVanityByTargetID(ctx, targetID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return toVanity(v), nil
}

func resolveByCode(ctx context.Context, code string) (*types.Vanity, error) {
	v, err := db.New(state.Pool).ResolveVanityByCode(ctx, code)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return toVanity(v), nil
}

func ResolveVanity(ctx context.Context, code string) (*types.Vanity, error) {
	q := db.New(state.Pool)

	botId, err := q.GetBotIDByClientID(ctx, code)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if botId != "" {
		return resolveByTargetID(ctx, botId)
	}

	serverId, err := q.GetServerIDByServerID(ctx, code)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if serverId != "" {
		return resolveByTargetID(ctx, serverId)
	}

	v, err := resolveByCode(ctx, code)

	if err != nil {
		return nil, err
	}

	if v != nil {
		return v, nil
	}

	return resolveByTargetID(ctx, code)
}

func ResolveVanityByItag(ctx context.Context, itag string) (*types.Vanity, error) {
	var itagUUID pgtype.UUID
	if err := itagUUID.Scan(itag); err != nil {
		return nil, nil
	}

	v, err := db.New(state.Pool).ResolveVanityByItag(ctx, itagUUID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return toVanity(v), nil
}
