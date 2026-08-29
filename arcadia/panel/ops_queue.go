// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}

	return &t.Time
}

type botQueueRow struct {
	BotID                string
	ClientID             string
	LastClaimed          *time.Time
	ClaimedBy            *string
	Type                 string
	ApprovalNote         string
	Short                string
	Invite               string
	ApproximateVotes     int32
	Shards               int32
	Library              string
	InviteClicks         int32
	Clicks               int32
	Servers              int32
	ModerationFlagged    bool
	ModerationCategories []string
}

func (s *Server) getUser(ctx context.Context, q *types.QGetUser) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	user, err := impls.GetPlatformUser(ctx, q.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, user), nil
}

func (s *Server) botQueue(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	rows, err := db.New(state.Pool).BotQueuePending(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	queue := make([]botQueueRow, len(rows))

	for i, row := range rows {
		var claimedBy *string

		if row.ClaimedBy.Valid {
			claimedBy = &row.ClaimedBy.String
		}

		queue[i] = botQueueRow{
			BotID:                row.BotID,
			ClientID:             row.ClientID,
			LastClaimed:          timePtr(row.LastClaimed),
			ClaimedBy:            claimedBy,
			Type:                 row.Type,
			ApprovalNote:         row.ApprovalNote,
			Short:                row.Short,
			Invite:               row.Invite,
			ApproximateVotes:     row.ApproximateVotes,
			Shards:               row.Shards,
			Library:              row.Library,
			InviteClicks:         row.InviteClicks,
			Clicks:               row.Clicks,
			Servers:              row.Servers,
			ModerationFlagged:    row.ModerationFlagged,
			ModerationCategories: row.ModerationCategories,
		}
	}

	return s.partialBots(ctx, queue)
}

func (s *Server) serverQueue(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	rows, err := db.New(state.Pool).ServerQueuePending(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	queue := make([]searchServerRow, len(rows))

	for i, row := range rows {
		var claimedBy *string

		if row.ClaimedBy.Valid {
			claimedBy = &row.ClaimedBy.String
		}

		queue[i] = searchServerRow{
			ServerID:             row.ServerID,
			Name:                 row.Name,
			Avatar:               row.Avatar,
			TotalMembers:         row.TotalMembers,
			OnlineMembers:        row.OnlineMembers,
			Short:                row.Short,
			Type:                 row.Type,
			ApprovalNote:         row.ApprovalNote,
			ApproximateVotes:     row.ApproximateVotes,
			InviteClicks:         row.InviteClicks,
			Clicks:               row.Clicks,
			NSFW:                 row.Nsfw,
			DiscordNSFWLevel:     int32(row.DiscordNsfwLevel),
			NSFWChannelCount:     row.NsfwChannelCount,
			Tags:                 row.Tags,
			Premium:              row.Premium,
			ClaimedBy:            claimedBy,
			LastClaimed:          timePtr(row.LastClaimed),
			ModerationFlagged:    row.ModerationFlagged,
			ModerationCategories: row.ModerationCategories,
		}
	}

	return s.partialServers(ctx, queue)
}

func (s *Server) partialServers(ctx context.Context, queue []searchServerRow) (response, error) {
	serverIDs := make([]string, 0, len(queue))

	for _, server := range queue {
		serverIDs = append(serverIDs, server.ServerID)
	}

	managers, err := impls.GetServerManagers(ctx, serverIDs)

	if err != nil {
		return response{}, newError(err)
	}

	servers := make([]types.PartialEntity, 0, len(queue))

	for _, server := range queue {
		servers = append(servers, types.PartialEntity{Server: &types.PartialServer{
			ServerID:             server.ServerID,
			Name:                 server.Name,
			Avatar:               server.Avatar,
			TotalMembers:         server.TotalMembers,
			OnlineMembers:        server.OnlineMembers,
			Short:                server.Short,
			Type:                 server.Type,
			ApprovalNote:         server.ApprovalNote,
			Votes:                server.ApproximateVotes,
			InviteClicks:         server.InviteClicks,
			Clicks:               server.Clicks,
			NSFW:                 server.NSFW,
			DiscordNSFWLevel:     server.DiscordNSFWLevel,
			NSFWChannelCount:     server.NSFWChannelCount,
			Tags:                 types.NonNilStrings(server.Tags),
			Premium:              server.Premium,
			ClaimedBy:            server.ClaimedBy,
			LastClaimed:          types.TimestampPtr(server.LastClaimed),
			Mentionable:          managers[server.ServerID].Mentionables(),
			ModerationFlagged:    server.ModerationFlagged,
			ModerationCategories: types.NonNilStrings(server.ModerationCategories),
		}})
	}

	return writeJSON(http.StatusOK, servers), nil
}

func (s *Server) partialBots(ctx context.Context, queue []botQueueRow) (response, error) {
	botIDs := make([]string, 0, len(queue))

	for _, bot := range queue {
		botIDs = append(botIDs, bot.BotID)
	}

	managers, err := impls.GetBotManagers(ctx, botIDs)

	if err != nil {
		return response{}, newError(err)
	}

	bots := make([]types.PartialEntity, 0, len(queue))

	for _, bot := range queue {
		user, err := impls.GetPlatformUser(ctx, bot.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		bots = append(bots, types.PartialEntity{Bot: &types.PartialBot{
			BotID:                bot.BotID,
			ClientID:             bot.ClientID,
			User:                 user,
			ClaimedBy:            bot.ClaimedBy,
			LastClaimed:          types.TimestampPtr(bot.LastClaimed),
			ApprovalNote:         bot.ApprovalNote,
			Short:                bot.Short,
			Type:                 bot.Type,
			Votes:                bot.ApproximateVotes,
			Shards:               bot.Shards,
			Library:              bot.Library,
			InviteClicks:         bot.InviteClicks,
			Clicks:               bot.Clicks,
			Servers:              bot.Servers,
			Mentionable:          managers[bot.BotID].Mentionables(),
			Invite:               bot.Invite,
			ModerationFlagged:    bot.ModerationFlagged,
			ModerationCategories: types.NonNilStrings(bot.ModerationCategories),
		}})
	}

	return writeJSON(http.StatusOK, bots), nil
}
