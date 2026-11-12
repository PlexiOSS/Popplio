package bot

import (
	"errors"
	"fmt"
	"net/http"

	"popplio/arcadia/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

var errArcadiaUnavailable = errors.New("Arcadia is not connected")

var arcadiaInGuild = func(guildID snowflake.ID) (bool, error) {
	if !dclient.Ready() {
		return false, errArcadiaUnavailable
	}

	arcadia := dclient.Get()

	if _, ok := arcadia.Caches.Guild(guildID); ok {
		return true, nil
	}

	_, err := arcadia.Rest.GetGuild(guildID, false)

	if err == nil {
		return true, nil
	}

	var restErr *rest.Error

	if errors.As(err, &restErr) && restErr.Response != nil {
		switch restErr.Response.StatusCode {
		case http.StatusForbidden, http.StatusNotFound:
			return false, nil
		}
	}

	return false, err
}

func omiAllowedIn(guildID snowflake.ID) (bool, error) {
	switch guildID {
	case state.Config.Servers.Main, state.Config.Servers.Staff, state.Config.Servers.Testing:
		return true, nil
	}

	return arcadiaInGuild(guildID)
}

var leaveGuild = func(client *bot.Client, guildID snowflake.ID) error {
	return client.Rest.LeaveGuild(guildID)
}

func guardOmiGuild(client *bot.Client, guildID snowflake.ID, name string) bool {
	allowed, err := omiAllowedIn(guildID)

	if err != nil {
		state.Logger.Warn("Couldn't tell whether Omi belongs in a server, staying", zap.Stringer("guild_id", guildID), zap.String("name", name), zap.Error(err))
		return false
	}

	if allowed {
		return false
	}

	if err := leaveGuild(client, guildID); err != nil {
		state.Logger.Error("Failed to leave an unofficial server", zap.Stringer("guild_id", guildID), zap.String("name", name), zap.Error(err))
		return false
	}

	state.Logger.Warn("Omi left a server that isn't one of ours", zap.Stringer("guild_id", guildID), zap.String("name", name))

	reportLeft(client, guildID, name)

	return true
}

var reportLeft = func(client *bot.Client, guildID snowflake.ID, name string) {
	if _, err := client.Rest.CreateMessage(state.Config.Channels.StaffLogs, discord.MessageCreate{
		Content:         fmt.Sprintf("Omi was added to **%s** (`%s`), which isn't one of our servers, and left it.", kbSafe(name), guildID),
		AllowedMentions: &discord.AllowedMentions{},
	}); err != nil {
		state.Logger.Warn("Failed to report leaving an unofficial server", zap.Error(err))
	}
}

func SweepOmiGuilds() {
	client := state.Discord

	if client == nil {
		return
	}

	var left int

	for guild := range client.Caches.Guilds() {
		if guardOmiGuild(client, guild.ID, guild.Name) {
			left++
		}
	}

	if left > 0 {
		state.Logger.Warn("Omi left servers that aren't ours on startup", zap.Int("count", left))
	}
}
