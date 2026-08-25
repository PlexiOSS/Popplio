package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

func onGuildsReady(ctx context.Context, _ *events.GuildsReady) {
	self, ok := dclient.Get().Caches().SelfUser()

	name := "arcadia"
	if ok {
		name = self.Username
	}

	state.Logger.Info(fmt.Sprintf("%s is ready! Doing some minor DB fixes", name))

	_, err := state.Pool.Exec(ctx, "UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE LOWER(claimed_by) = 'none'")

	if err != nil {
		state.Logger.Error("Failed to run startup DB fixes", zap.Error(err))
	}

	if err := SyncCommands(); err != nil {
		state.Logger.Error("Failed to sync application commands", zap.Error(err))
	}
}

func onGuildMemberJoin(e *events.GuildMemberJoin) {
	if e.GuildID != state.Config.Servers.Main {
		return
	}

	member := e.Member
	face := member.User.EffectiveAvatarURL()
	now := time.Now()

	if member.User.Bot {
		err := impls.SendChannel(state.Config.Channels.System, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:     "__**New Bot Added**__",
				Color:     impls.ColourGreen,
				Thumbnail: &discord.EmbedResource{URL: face},
				Timestamp: &now,
				Description: fmt.Sprintf(
					"Bot <@%s> (%s) has been added to the server.", member.User.ID, member.User.Username),
			}},
		})

		if err != nil {
			state.Logger.Error("Failed to announce new bot", zap.Error(err))
		}

		err = impls.AddRole(state.Config.Servers.Main, member.User.ID, state.Config.Roles.BotRole, "Bot added to server")

		if err != nil {
			state.Logger.Error("Failed to add bot role", zap.Error(err))
		}

		return
	}

	err := impls.SendChannel(state.Config.Channels.System, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:     "__**New User**__",
			Color:     impls.ColourGreen,
			Thumbnail: &discord.EmbedResource{URL: face},
			Timestamp: &now,
			Description: fmt.Sprintf(
				"Hmmmm... looks like <@%s> (%s) has strolled in. Can we trust them?", member.User.ID, member.User.Username),
		}},
	})

	if err != nil {
		state.Logger.Error("Failed to announce new user", zap.Error(err))
	}
}

func mainServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Main {
		return errors.New("You are not in the main server")
	}

	return nil
}

func staffServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Staff {
		return errors.New("You are not in the staff server")
	}

	return nil
}

func testingServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Testing {
		return errors.New("You are not in the testing server")
	}

	return nil
}

// mainOrTestingServer restricts a command to the two community guilds,
// excluding the staff server. Kept as one check rather than teaching Checks
// to OR, since every command that needs it needs exactly this pair.
func mainOrTestingServer(c *Ctx) error {
	if c.GuildID != state.Config.Servers.Main && c.GuildID != state.Config.Servers.Testing {
		return errors.New("this command can only be used in the main or testing server")
	}

	return nil
}

func isStaff(c *Ctx) error {
	if perms.IsConfigOwner(c.Author.ID.String()) {
		return nil
	}

	var count int64

	err := state.Pool.QueryRow(c.Context, "SELECT COUNT(*) FROM staff_members WHERE user_id = $1", c.Author.ID.String()).Scan(&count)

	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("You are not staff")
	}

	return nil
}
