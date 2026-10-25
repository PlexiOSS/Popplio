package bot

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"popplio/arcadia/impls"
	"popplio/arcadia/tasks"
	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

func cmdAnalytics() *Command {
	return &Command{
		Name:        "analytics",
		Category:    "Stats",
		Description: "Look at our site analytics!",
		Run: func(c *Ctx) error {
			byType := map[string]int64{}

			q := db.New(state.Pool)

			typeRows, err := q.CountBotsByType(c.Context)

			if err != nil {
				return err
			}

			for _, row := range typeRows {
				byType[row.Method] = row.Count
			}

			counts, err := q.GetAnalyticsCounts(c.Context)

			if err != nil {
				return err
			}

			bots, teams, users, guilds, packs := counts.Bots, counts.Teams, counts.Users, counts.Servers, counts.Packs

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Omniplex Analytics",
					Description: "I hope it's good :eyes:",
					Color:       impls.ColourGreen,
					Fields: []discord.EmbedField{
						{Name: "User Count:", Value: strconv.FormatInt(users, 10), Inline: impls.InlineTrue()},
						{Name: "Team Count:", Value: strconv.FormatInt(teams, 10), Inline: impls.InlineTrue()},
						{Name: "Bot Count:", Value: strconv.FormatInt(bots, 10), Inline: impls.InlineTrue()},
						{Name: "Server Count:", Value: strconv.FormatInt(guilds, 10), Inline: impls.InlineTrue()},
						{Name: "Pack Count:", Value: strconv.FormatInt(packs, 10), Inline: impls.InlineTrue()},
						{Name: "Approved Bots:", Value: strconv.FormatInt(byType["approved"], 10), Inline: impls.InlineTrue()},
						{Name: "Denied Bots:", Value: strconv.FormatInt(byType["denied"], 10), Inline: impls.InlineTrue()},
						{Name: "Certified Bots:", Value: strconv.FormatInt(byType["certified"], 10), Inline: impls.InlineTrue()},
						{Name: "Test Bots (hidden):", Value: strconv.FormatInt(byType["testbot"], 10), Inline: impls.InlineTrue()},
					},
				}},
			})
		},
	}
}

func cmdInfo() *Command {
	return &Command{
		Name:        "info",
		Category:    "Stats",
		Description: "Bot information",
		Run: func(c *Ctx) error {
			var (
				revision = "unknown"
				modified = "unknown"
				version  = "unknown"
			)

			if bi, ok := debug.ReadBuildInfo(); ok {
				version = bi.Main.Version

				for _, setting := range bi.Settings {
					switch setting.Key {
					case "vcs.revision":
						revision = setting.Value
					case "vcs.modified":
						modified = setting.Value
					}
				}
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title: "Bot Information:",
					Color: impls.ColourGreen,
					Fields: []discord.EmbedField{
						{Name: "Bot Version:", Value: version, Inline: impls.InlineTrue()},
						{Name: "Go Version:", Value: runtime.Version(), Inline: impls.InlineTrue()},
						{Name: "Git Commit:", Value: revision, Inline: impls.InlineTrue()},
						{Name: "Modified:", Value: modified, Inline: impls.InlineTrue()},
						{Name: "Built On:", Value: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), Inline: impls.InlineTrue()},
					},
				}},
			})
		},
	}
}

func cmdLeaderboard() *Command {
	return &Command{
		Name:        "leaderboard",
		Category:    "Leaderboard",
		Description: "Top reviewers by approvals and denials",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{Name: "limit", Description: "Limit the amount of results."},
		},
		Run: func(c *Ctx) error {
			limit := 5

			if raw := c.Option("limit", 0); raw != "" {
				parsed, err := strconv.ParseInt(raw, 10, 32)

				if err == nil && parsed > 0 {
					limit = int(min(parsed, 25))
				}
			}

			stats, err := tasks.QueryTopReviewers(c.Context, limit)

			if err != nil {
				return err
			}

			var sb strings.Builder

			sb.WriteString("Oh, hello there! Let's see who's been fighting bots the most :eyes:\n\n")

			for i, stat := range stats {
				fmt.Fprintf(&sb, "%s <@%s> | Approved: %d | Denied: %d | Total: **%d**\n",
					medal(i), stat.UserID, stat.ApprovedCount, stat.DeniedCount, stat.TotalCount)
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "Staff Leaderboard",
					Description: sb.String(),
					Color:       impls.ColourGreen,
				}},
			})
		},
	}
}

func medal(index int) string {
	switch index {
	case 0:
		return "🥇"
	case 1:
		return "🥈"
	case 2:
		return "🥉"
	default:
		return fmt.Sprintf("%d.", index+1)
	}
}
