// Copyright (C) 2026 NodeByte LTD

package fetchers

import (
	"context"
	"fmt"
	"time"

	"github.com/PlexiOSS/Keel/uuidutil"
	"popplio/db"
	"popplio/seo"
	"popplio/state"

	"github.com/PlexiOSS/Keel/dovewing"
)

type TeamFetcher struct{}

func (t *TeamFetcher) Type() string {
	return "team"
}

func (t *TeamFetcher) Fetch(ctx context.Context, mg *seo.MapGenerator, id string) (*seo.Entity, error) {
	row, err := db.New(state.Pool).GetTeamSEOFields(ctx, id)

	if err != nil {
		return nil, err
	}

	return &seo.Entity{
		ID:   id,
		Type: t.Type(),
		Name: row.Name,
		Description: func() string {
			if row.Short.Valid {
				return row.Short.String
			}

			return "This team seems to be a bit mysterious indeed!"
		}(),
		URL:       fmt.Sprintf("%s/teams/%s", state.Config.Sites.Frontend, id),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// Fetcher for a user
type UserFetcher struct{}

func (u *UserFetcher) Type() string {
	return "user"
}

func (u *UserFetcher) Fetch(ctx context.Context, mg *seo.MapGenerator, id string) (*seo.Entity, error) {
	q := db.New(state.Pool)

	exists, err := q.UserExists(ctx, id)

	if err != nil {
		return nil, err
	}

	pu, err := dovewing.GetUser(ctx, id, state.DovewingPlatformDiscord)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !exists {
		return &seo.Entity{
			ID:          id,
			Type:        u.Type(),
			AvatarURL:   pu.Avatar,
			Name:        pu.Username,
			Description: "This user seems to be on a distant island somewhere???",
			URL:         fmt.Sprintf("%s/user/%s", state.Config.Sites.Frontend, id),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	row, err := q.GetUserAboutAndTimestamps(ctx, id)

	if err != nil {
		return nil, err
	}

	return &seo.Entity{
		ID:          id,
		Type:        u.Type(),
		AvatarURL:   pu.Avatar,
		Name:        pu.Username,
		Description: row.About.String,
		URL:         fmt.Sprintf("%s/user/%s", state.Config.Sites.Frontend, id),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

type BotFetcher struct{}

func (b *BotFetcher) Type() string {
	return "bot"
}

func (b *BotFetcher) Fetch(ctx context.Context, mg *seo.MapGenerator, id string) (*seo.Entity, error) {
	row, err := db.New(state.Pool).GetBotSEOFields(ctx, id)

	if err != nil {
		return nil, err
	}

	botUser, err := dovewing.GetUser(ctx, id, state.DovewingPlatformDiscord)

	if err != nil {
		return nil, fmt.Errorf("failed to get bot user: %w", err)
	}

	var resolvedOwner *seo.Entity

	if row.TeamOwner.Valid {
		resolvedOwner, err = mg.Add(ctx, &TeamFetcher{}, uuidutil.Encode(row.TeamOwner.Bytes))

		if err != nil {
			return nil, fmt.Errorf("failed to resolve team owner: %w", err)
		}
	}

	if row.Owner.Valid {
		resolvedOwner, err = mg.Add(ctx, &UserFetcher{}, row.Owner.String)

		if err != nil {
			return nil, fmt.Errorf("failed to resolve owner: %w", err)
		}
	}

	return &seo.Entity{
		ID:          id,
		Type:        b.Type(),
		Name:        botUser.Username,
		AvatarURL:   botUser.Avatar,
		Description: row.Short,
		URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, id),
		Author:      resolvedOwner,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}
