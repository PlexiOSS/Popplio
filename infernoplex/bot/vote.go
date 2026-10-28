// Copyright (C) 2026 NodeByte LTD

package bot

import (
	"errors"
	"fmt"

	"popplio/captcha"
	"popplio/db"
	"popplio/state"
	"popplio/votes/cast"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func cmdVote() *Command {
	return &Command{
		Name:        "vote",
		Description: "Vote for this server on Omniplex",
		Run: func(c *Ctx) error {
			if c.GuildID == 0 {
				return errors.New("This command can only be used in a server")
			}

			if err := c.Defer(); err != nil {
				return err
			}

			serverID := c.GuildID.String()
			userID := c.Author.ID.String()
			serverURL := state.Config.Sites.Frontend + "/servers/" + serverID
			linkRow := discord.NewActionRow(discord.NewLinkButton("Vote on Omniplex", serverURL))

			q := db.New(state.Pool)

			banStatus, err := q.GetUserBanStatus(c.Context, userID)

			if errors.Is(err, pgx.ErrNoRows) {
				return c.Send(discord.MessageCreate{
					Embeds: []discord.Embed{{
						Description: "Sign in to Omniplex once with Discord, then you can vote from here.",
						Color:       colourBlurple,
					}},
					Components: []discord.ContainerComponent{linkRow},
				})
			}

			if err != nil {
				state.Logger.Error("Failed to load voter", zap.Error(err), zap.String("user_id", userID))
				return errors.New("Something went wrong. Please try again in a moment.")
			}

			if banStatus.Banned {
				return c.Fail("Your Omniplex account is banned, so you can't vote.")
			}

			voteBanned, err := q.GetUserVoteBanned(c.Context, userID)

			if err != nil {
				state.Logger.Error("Failed to check vote ban", zap.Error(err), zap.String("user_id", userID))
				return errors.New("Something went wrong. Please try again in a moment.")
			}

			if voteBanned {
				return c.Fail("You're banned from voting right now. Contact support if you think this is a mistake.")
			}

			needsCaptcha, err := captcha.RequiresCaptcha(c.Context, "server", serverID)

			if err != nil {
				state.Logger.Error("Failed to check captcha requirement", zap.Error(err), zap.String("server_id", serverID))
				return errors.New("Something went wrong. Please try again in a moment.")
			}

			if needsCaptcha {
				return c.Send(discord.MessageCreate{
					Embeds: []discord.Embed{{
						Description: "Votes for this server need a quick check on the website first.",
						Color:       colourBlurple,
					}},
					Components: []discord.ContainerComponent{linkRow},
				})
			}

			res, err := cast.Cast(c.Context, userID, "server", serverID, true, false)

			if err != nil {
				var castErr *cast.Error

				if errors.As(err, &castErr) {
					return c.Fail(castErr.Message)
				}

				state.Logger.Error("Failed to cast vote from Discord", zap.Error(err), zap.String("server_id", serverID), zap.String("user_id", userID))
				return errors.New("Something went wrong casting your vote. Please try again in a moment.")
			}

			cast.Announce(userID, "server", serverID, res, false)

			name := "this server"
			if res.Entity != nil && res.Entity.Name != "" {
				name = "**" + res.Entity.Name + "**"
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Thanks for voting!",
					Description: fmt.Sprintf("Your vote for %s counted as %d. You can vote again once your cooldown ends.", name, res.Votes),
					Color:       colourGreen,
				}},
				Components: []discord.ContainerComponent{
					discord.NewActionRow(discord.NewLinkButton("View on Omniplex", serverURL)),
				},
			})
		},
	}
}
