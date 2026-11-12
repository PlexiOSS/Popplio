// Copyright (C) 2026 NodeByte LTD

package cmdsync

import (
	"errors"
	"fmt"
	"net/http"

	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

type Plan struct {
	Name   string
	Global []discord.ApplicationCommandCreate
	Guilds map[snowflake.ID][]discord.ApplicationCommandCreate
}

func Sync(client *bot.Client, plan Plan) error {
	appID := client.ApplicationID
	log := state.Logger.With(zap.String("bot", plan.Name))
	var errs []error

	global := plan.Global
	if global == nil {
		global = []discord.ApplicationCommandCreate{}
	}

	if _, err := client.Rest.SetGlobalCommands(appID, global); err != nil {
		errs = append(errs, fmt.Errorf("setting global commands: %w", err))
	} else {
		log.Info("Synced global slash commands", zap.Int("count", len(global)))
	}

	for guildID, cmds := range plan.Guilds {
		if guildID == 0 {
			continue
		}

		if cmds == nil {
			cmds = []discord.ApplicationCommandCreate{}
		}

		if _, err := client.Rest.SetGuildCommands(appID, guildID, cmds); err != nil {
			if noAccess(err) {
				log.Warn("Not in a guild it should have commands in, skipping", zap.Stringer("guild_id", guildID))
				continue
			}

			errs = append(errs, fmt.Errorf("setting commands in guild %s: %w", guildID, err))
			continue
		}

		log.Info("Synced guild slash commands", zap.Stringer("guild_id", guildID), zap.Int("count", len(cmds)))
	}

	var others []snowflake.ID

	for g := range client.Caches.Guilds() {
		if _, planned := plan.Guilds[g.ID]; !planned {
			others = append(others, g.ID)
		}
	}

	for _, guildID := range others {
		existing, err := client.Rest.GetGuildCommands(appID, guildID, false)

		if err != nil {
			if !noAccess(err) {
				errs = append(errs, fmt.Errorf("reading commands in guild %s: %w", guildID, err))
			}

			continue
		}

		if len(existing) == 0 {
			continue
		}

		if _, err := client.Rest.SetGuildCommands(appID, guildID, []discord.ApplicationCommandCreate{}); err != nil {
			errs = append(errs, fmt.Errorf("clearing commands in guild %s: %w", guildID, err))
			continue
		}

		log.Info("Removed leftover guild slash commands", zap.Stringer("guild_id", guildID), zap.Int("count", len(existing)))
	}

	return errors.Join(errs...)
}

func noAccess(err error) bool {
	var resp *http.Response

	if ptr := (*rest.Error)(nil); errors.As(err, &ptr) && ptr != nil {
		resp = ptr.Response
	}

	return resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
}
