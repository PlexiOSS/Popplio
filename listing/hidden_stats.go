// Copyright (C) 2026 NodeByte LTD

package listing

import (
	"context"
	"errors"
	"maps"
	"slices"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	StatServers = "servers"
	StatShards  = "shards"
	StatUptime  = "uptime"
	StatViews   = "views"
	StatInvites = "invites"
	StatMembers = "members"
	StatBoosts  = "boosts"
)

var (
	BotHideableStats    = []string{StatServers, StatShards, StatUptime, StatViews, StatInvites}
	ServerHideableStats = []string{StatMembers, StatBoosts, StatViews, StatInvites}
)

func CleanHiddenStats(allowed, requested []string) ([]string, bool) {
	for _, stat := range requested {
		if !slices.Contains(allowed, stat) {
			return nil, false
		}
	}

	out := make([]string, 0, len(requested))

	for _, stat := range allowed {
		if slices.Contains(requested, stat) {
			out = append(out, stat)
		}
	}

	return out, true
}

func withoutStatus(user *dovetypes.PlatformUser) *dovetypes.PlatformUser {
	if user == nil {
		return nil
	}

	out := *user
	out.Status = ""
	out.ExtraData = maps.Clone(user.ExtraData)
	delete(out.ExtraData, "presence_unknown")

	return &out
}

func HideBotUserStatus(ctx context.Context, botID string, user *dovetypes.PlatformUser) *dovetypes.PlatformUser {
	hidden, err := db.New(state.Pool).GetBotHiddenStats(ctx, botID)

	if errors.Is(err, pgx.ErrNoRows) {
		return user
	}

	if err != nil || slices.Contains(hidden, StatUptime) {
		return withoutStatus(user)
	}

	return user
}

func HideBotStats(bot *types.Bot) {
	if slices.Contains(bot.HiddenStats, StatServers) {
		bot.Servers = 0
	}

	if slices.Contains(bot.HiddenStats, StatShards) {
		bot.Shards = 0
		bot.ShardList = []int{}
	}

	if slices.Contains(bot.HiddenStats, StatUptime) {
		bot.Uptime = 0
		bot.TotalUptime = 0
		bot.UptimePercent = nil
		bot.UptimeChecks = 0
		bot.UptimeLastChecked = pgtype.Timestamptz{}
		bot.User = withoutStatus(bot.User)
	}

	if slices.Contains(bot.HiddenStats, StatViews) {
		bot.Clicks = 0
		bot.UniqueClicks = 0
	}

	if slices.Contains(bot.HiddenStats, StatInvites) {
		bot.InviteClicks = 0
	}
}

func HideIndexBotStats(bot *types.IndexBot) {
	if slices.Contains(bot.HiddenStats, StatUptime) {
		bot.User = withoutStatus(bot.User)
	}

	if slices.Contains(bot.HiddenStats, StatServers) {
		bot.Servers = 0
	}

	if slices.Contains(bot.HiddenStats, StatShards) {
		bot.Shards = 0
	}

	if slices.Contains(bot.HiddenStats, StatViews) {
		bot.Clicks = 0
	}

	if slices.Contains(bot.HiddenStats, StatInvites) {
		bot.InviteClick = 0
	}
}

func HideServerStats(server *types.Server) {
	if slices.Contains(server.HiddenStats, StatMembers) {
		server.TotalMembers = 0
		server.OnlineMembers = 0
	}

	if slices.Contains(server.HiddenStats, StatBoosts) {
		server.PremiumTier = 0
		server.BoostCount = 0
	}

	if slices.Contains(server.HiddenStats, StatViews) {
		server.Clicks = 0
		server.UniqueClicks = 0
	}

	if slices.Contains(server.HiddenStats, StatInvites) {
		server.InviteClicks = 0
	}
}

func HideIndexServerStats(server *types.IndexServer) {
	if slices.Contains(server.HiddenStats, StatMembers) {
		server.TotalMembers = 0
		server.OnlineMembers = 0
	}

	if slices.Contains(server.HiddenStats, StatViews) {
		server.Clicks = 0
	}

	if slices.Contains(server.HiddenStats, StatInvites) {
		server.InviteClicks = 0
	}
}
