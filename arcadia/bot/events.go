package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/botroles"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

func onGuildsReady(ctx context.Context, _ *events.GuildsReady) {
	self, ok := dclient.Get().Caches.SelfUser()

	name := "arcadia"
	if ok {
		name = self.Username
	}

	state.Logger.Info(fmt.Sprintf("%s is ready! Doing some minor DB fixes", name))

	err := db.New(state.Pool).FixNoneClaimedBots(ctx)

	if err != nil {
		state.Logger.Error("Failed to run startup DB fixes", zap.Error(err))
	}

	if err := SyncArcadiaCommands(); err != nil {
		state.Logger.Error("Failed to sync application commands", zap.Error(err))
	}
}

func onGuildMemberJoin(e *events.GuildMemberJoin) {
	if e.GuildID != state.Config.Servers.Main {
		return
	}

	d := impls.With(e.Client())

	member := e.Member
	face := member.User.EffectiveAvatarURL()
	now := time.Now()

	if member.User.Bot {
		listing := botroles.Listing{}

		if found, err := botroles.Lookup(context.Background(), []string{member.User.ID.String()}); err != nil {
			state.Logger.Error("Failed to look up a new bot's listing", zap.Error(err))
		} else {
			listing = found[member.User.ID.String()]
		}

		kind := "Bot"

		switch {
		case listing.Partner && listing.Certified():
			kind = "Certified partner bot"
		case listing.Partner:
			kind = "Partner bot"
		case listing.Certified():
			kind = "Certified bot"
		}

		description := fmt.Sprintf("%s <@%s> (%s) has been added to the server.", kind, member.User.ID, member.User.Username)

		if listing.Listed() {
			description += fmt.Sprintf("\n[View on Omniplex](%s/bots/%s)", frontendURL(), member.User.ID)
		} else {
			description += "\nIt isn't listed on Omniplex."
		}

		err := d.SendChannel(state.Config.Channels.System, discord.MessageCreate{
			Embeds: []discord.Embed{{
				Title:       "__**New Bot Added**__",
				Color:       impls.ColourGreen,
				Thumbnail:   &discord.EmbedResource{URL: face},
				Timestamp:   &now,
				Description: description,
			}},
		})

		if err != nil {
			state.Logger.Error("Failed to announce new bot", zap.Error(err))
		}

		if err := botroles.Give(e.Client(), member.User.ID, botroles.Missing(member.RoleIDs, botroles.Wanted(listing)), "Bot added to server"); err != nil {
			state.Logger.Error("Failed to give a new bot its roles", zap.Error(err))
		}

		return
	}

	err := d.SendChannel(state.Config.Channels.System, discord.MessageCreate{
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

	count, err := db.New(state.Pool).CountStaffMemberByID(c.Context, c.Author.ID.String())

	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("You are not staff")
	}

	return nil
}
