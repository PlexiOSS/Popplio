// Copyright (C) 2026 NodeByte LTD

package servervotes

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"popplio/db"
	"popplio/infernoplex/dclient"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

var ErrUnavailable = errors.New("the Infernoplex bot isn't connected right now")

func Channels(serverID string) ([]types.ServerVoteChannelOption, error) {
	if !dclient.Ready() {
		return nil, ErrUnavailable
	}

	guildID, err := snowflake.Parse(serverID)

	if err != nil {
		return nil, err
	}

	channels, err := dclient.Get().Rest().GetGuildChannels(guildID)

	if err != nil {
		return nil, err
	}

	out := make([]types.ServerVoteChannelOption, 0, len(channels))

	for _, c := range channels {
		switch c.Type() {
		case discord.ChannelTypeGuildText, discord.ChannelTypeGuildNews:
			out = append(out, types.ServerVoteChannelOption{ID: c.ID().String(), Name: c.Name()})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out, nil
}

func SendTest(channelID string) error {
	if !dclient.Ready() {
		return ErrUnavailable
	}

	id, err := snowflake.Parse(channelID)

	if err != nil {
		return err
	}

	_, err = dclient.Get().Rest().CreateMessage(id, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Vote messages are on",
			Description: "New votes on Omniplex will be posted in this channel.",
			Color:       0x8A6BFD,
		}},
	})

	return err
}

func Post(ctx context.Context, serverID, voter string, votes int, auto bool, listingURL, name string) {
	channelID, err := db.New(state.Pool).GetServerVoteChannel(ctx, serverID)

	if err != nil || !channelID.Valid || !dclient.Ready() {
		return
	}

	id, err := snowflake.Parse(channelID.String)

	if err != nil {
		return
	}

	description := fmt.Sprintf("**%s** voted for **%s**. It now has **%d** votes.", voter, name, votes)

	if auto {
		description = fmt.Sprintf("**%s** voted for **%s** with Auto-vote. It now has **%d** votes.", voter, name, votes)
	}

	if _, err := dclient.Get().Rest().CreateMessage(id, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "New vote",
			URL:         listingURL,
			Description: description + fmt.Sprintf("\n\n[Vote on Omniplex](%s)", listingURL),
			Color:       0x8A6BFD,
		}},
	}); err != nil {
		state.Logger.Warn("Failed to post a server vote message", zap.Error(err), zap.String("server_id", serverID), zap.String("channel_id", channelID.String))
	}
}
