// Copyright (C) 2026 NodeByte LTD

package listing

import (
	"slices"
	"testing"

	"popplio/types"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCleanHiddenStats(t *testing.T) {
	got, ok := CleanHiddenStats(BotHideableStats, []string{StatViews, StatServers, StatViews})

	if !ok || !slices.Equal(got, []string{StatServers, StatViews}) {
		t.Fatalf("got %v, %v", got, ok)
	}

	if _, ok := CleanHiddenStats(BotHideableStats, []string{"votes"}); ok {
		t.Fatal("votes can't be hidden")
	}

	if _, ok := CleanHiddenStats(BotHideableStats, []string{StatMembers}); ok {
		t.Fatal("a bot has no members stat")
	}

	if got, ok := CleanHiddenStats(ServerHideableStats, nil); !ok || got == nil || len(got) != 0 {
		t.Fatalf("clearing = %v, %v", got, ok)
	}
}

func TestHideBotStats(t *testing.T) {
	percent := 99.0
	shared := &dovetypes.PlatformUser{
		ID:        "1",
		Status:    dovetypes.PlatformStatusOnline,
		ExtraData: map[string]any{"presence_unknown": true},
	}
	bot := types.Bot{
		Servers: 10, Shards: 2, ShardList: []int{1, 2}, Uptime: 5, TotalUptime: 6,
		UptimePercent: &percent, UptimeChecks: 6, UptimeLastChecked: pgtype.Timestamptz{Valid: true},
		Clicks: 7, Views30d: 5, UniqueClicks: 3, InviteClicks: 4, ApproximateVotes: 9,
		HiddenStats: BotHideableStats,
		User:        shared,
	}

	HideBotStats(&bot)

	if bot.User.Status != "" || bot.User.ExtraData["presence_unknown"] != nil {
		t.Fatalf("status leaked: %+v", bot.User)
	}

	if shared.Status != dovetypes.PlatformStatusOnline || shared.ExtraData["presence_unknown"] != true {
		t.Fatalf("the shared user was changed: %+v", shared)
	}

	if bot.Servers != 0 || bot.Shards != 0 || len(bot.ShardList) != 0 || bot.Uptime != 0 || bot.TotalUptime != 0 ||
		bot.UptimePercent != nil || bot.UptimeChecks != 0 || bot.UptimeLastChecked.Valid || bot.Clicks != 0 || bot.Views30d != 0 ||
		bot.UniqueClicks != 0 || bot.InviteClicks != 0 {
		t.Fatalf("something leaked: %+v", bot)
	}

	if bot.ApproximateVotes != 9 {
		t.Fatal("votes must stay")
	}

	partial := types.Bot{Servers: 10, Clicks: 7, HiddenStats: []string{StatServers}, User: shared}
	HideBotStats(&partial)

	if partial.Servers != 0 || partial.Clicks != 7 || partial.User.Status != dovetypes.PlatformStatusOnline {
		t.Fatalf("partial = %+v", partial)
	}
}

func TestHideServerStats(t *testing.T) {
	server := types.Server{
		TotalMembers: 10, OnlineMembers: 5, PremiumTier: 2, BoostCount: 3, Clicks: 7, UniqueClicks: 2, InviteClicks: 4,
		HiddenStats: ServerHideableStats,
	}

	HideServerStats(&server)

	if server.TotalMembers != 0 || server.OnlineMembers != 0 || server.PremiumTier != 0 || server.BoostCount != 0 ||
		server.Clicks != 0 || server.UniqueClicks != 0 || server.InviteClicks != 0 {
		t.Fatalf("something leaked: %+v", server)
	}
}

func TestHideIndexStats(t *testing.T) {
	user := &dovetypes.PlatformUser{ID: "1", Status: dovetypes.PlatformStatusOnline}
	bot := types.IndexBot{Servers: 1, Shards: 1, Clicks: 1, InviteClick: 1, HiddenStats: BotHideableStats, User: user}
	HideIndexBotStats(&bot)

	if bot.Servers != 0 || bot.Shards != 0 || bot.Clicks != 0 || bot.InviteClick != 0 || bot.User.Status != "" {
		t.Fatalf("bot leaked: %+v", bot)
	}

	server := types.IndexServer{TotalMembers: 1, OnlineMembers: 1, Clicks: 1, InviteClicks: 1, HiddenStats: ServerHideableStats}
	HideIndexServerStats(&server)

	if server.TotalMembers != 0 || server.OnlineMembers != 0 || server.Clicks != 0 || server.InviteClicks != 0 {
		t.Fatalf("server leaked: %+v", server)
	}
}
