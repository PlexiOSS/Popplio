// Copyright (C) 2026 NodeByte LTD

package state

import (
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

func BotPresence(activity string) gateway.ConfigOpt {
	return gateway.WithPresenceOpts(
		gateway.WithOnlineStatus(discord.OnlineStatusOnline),
		gateway.WithWatchingActivity(activity),
	)
}

func SiteHost() string {
	host := strings.TrimPrefix(strings.TrimPrefix(Config.Sites.Frontend, "https://"), "http://")
	return strings.TrimSuffix(host, "/")
}
