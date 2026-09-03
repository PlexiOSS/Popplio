// Copyright (C) 2026 NodeByte LTD

package hooks

import (
	"errors"
	"fmt"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/core/events"
	"popplio/webhooks/sender"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type ServerDriver struct{}

func (sd ServerDriver) TargetType() string {
	return "server"
}

func (sd ServerDriver) Construct(userId, id string) (*events.Target, *sender.WebhookEntity, error) {
	q := db.New(state.Pool)

	row, err := q.GetIndexServerByID(state.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errors.New("server not found")
	}

	if err != nil {
		state.Logger.Error("Failed to fetch server for this hook", zap.Error(err), zap.String("serverID", id), zap.String("userID", userId))
		return nil, nil, err
	}

	server := types.IndexServer{
		ServerID:         row.ServerID,
		Name:             row.Name,
		Avatar:           row.Avatar,
		TotalMembers:     int(row.TotalMembers),
		OnlineMembers:    int(row.OnlineMembers),
		Short:            row.Short,
		Type:             row.Type,
		State:            row.State,
		VanityRef:        row.VanityRef,
		ApproximateVotes: int(row.ApproximateVotes),
		InviteClicks:     int(row.InviteClicks),
		Clicks:           int(row.Clicks),
		NSFW:             row.Nsfw,
		Tags:             row.Tags,
		Premium:          row.Premium,
		SupporterBadge:   row.SupporterBadge,
		BoostedUntil:     row.BoostedUntil,
		FeaturedUntil:    row.FeaturedUntil,
		SpotlightedUntil: row.SpotlightedUntil,
	}

	code, err := q.GetVanityCodeByItag(state.Context, server.VanityRef)

	if err != nil {
		return nil, nil, fmt.Errorf("error while getting server vanity code [db fetch]: %w", err)
	}

	server.Vanity = code

	targets := events.Target{
		Server: &server,
	}

	entity := sender.WebhookEntity{
		EntityID:   server.ServerID,
		EntityName: server.Name,
		EntityType: sd.TargetType(),
	}

	return &targets, &entity, nil
}

func (sd ServerDriver) CanBeConstructed(userId, targetId string) (bool, error) {
	return true, nil
}

func (sd ServerDriver) SupportsPullPending(userId, targetId string) (bool, error) {
	return true, nil
}

func init() {
	drivers.RegisterDriver(ServerDriver{})
}
