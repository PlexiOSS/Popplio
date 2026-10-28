// Copyright (C) 2026 NodeByte LTD

package bot

import (
	"errors"
	"fmt"
	"strings"

	"popplio/bumps"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
)

func cmdBump() *Command {
	return &Command{
		Name:        "bump",
		Description: "Bump this server to the top of Omniplex's Recently bumped list (every 2 hours)",
		Run: func(c *Ctx) error {
			if c.GuildID == 0 {
				return errors.New("This command can only be used in a server")
			}

			serverID := c.GuildID.String()
			serverURL := state.Config.Sites.Frontend + "/servers/" + serverID

			next, err := bumps.Bump(c.Context, serverID, c.Author.ID.String())

			var cooldown *bumps.CooldownError

			switch {
			case errors.Is(err, bumps.ErrNotFound):
				return c.Fail(fmt.Sprintf("This server isn't listed on Omniplex yet. An admin can add it at %s/servers/add.", state.Config.Sites.Frontend))
			case errors.Is(err, bumps.ErrNotPublic):
				return c.Fail("Only public servers that have been approved can be bumped.")
			case errors.As(err, &cooldown):
				return c.Fail(fmt.Sprintf("This server was bumped recently. You can bump it again <t:%d:R>.", cooldown.Next.Unix()))
			case err != nil:
				state.Logger.Error("Failed to bump server", zap.Error(err), zap.String("server_id", serverID))
				return errors.New("Something went wrong bumping this server. Please try again in a moment.")
			}

			description := fmt.Sprintf("<@%s> bumped this server to the top of Omniplex's **Recently bumped** list. You can bump it again <t:%d:R>.", c.Author.ID, next.Unix())

			if info, err := bumps.Info(c.Context, serverID, 3); err == nil && len(info.TopBumpers) > 0 {
				lines := make([]string, 0, len(info.TopBumpers))

				for i, b := range info.TopBumpers {
					lines = append(lines, fmt.Sprintf("%d. <@%s> · %d", i+1, b.User.ID, b.Bumps))
				}

				description += "\n\n**Top bumpers (30 days)**\n" + strings.Join(lines, "\n")
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Bumped!",
					Description: description,
					Color:       colourGreen,
				}},
				Components: []discord.ContainerComponent{
					discord.NewActionRow(discord.NewLinkButton("View on Omniplex", serverURL)),
				},
				AllowedMentions: &discord.AllowedMentions{},
			})
		},
	}
}
