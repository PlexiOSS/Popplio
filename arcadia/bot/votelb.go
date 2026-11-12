package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"

	db "popplio/db"
	"popplio/state"
	"popplio/votes"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

const (
	voteBoardBots    = "bots"
	voteBoardServers = "servers"
	voteBoardVoters  = "voters"
	voteBoardSize    = 10
	voteBoardPrefix  = "lb:"
)

var voteBoardTitles = map[string]string{
	voteBoardBots:    "Top bots this month",
	voteBoardServers: "Top servers this month",
	voteBoardVoters:  "Top voters this month",
}

type voteBoardRow struct {
	name  string
	url   string
	votes int
	extra string
}

var loadVoteBoard = func(ctx context.Context, kind string) ([]voteBoardRow, error) {
	q := db.New(state.Pool)

	switch kind {
	case voteBoardServers:
		rows, err := q.GetTopVotedServers(ctx, voteBoardSize)

		if err != nil {
			return nil, err
		}

		out := make([]voteBoardRow, len(rows))

		for i, r := range rows {
			out[i] = voteBoardRow{
				name:  r.Name,
				url:   fmt.Sprintf("%s/servers/%s", frontendURL(), r.ServerID),
				votes: int(r.ApproximateVotes),
			}
		}

		return out, nil
	case voteBoardVoters:
		entries, err := votes.Leaderboard(ctx, votes.LeaderboardPeriodMonth, voteBoardSize)

		if err != nil {
			return nil, err
		}

		out := make([]voteBoardRow, len(entries))

		for i, e := range entries {
			row := voteBoardRow{name: "Unknown user", votes: e.Votes}

			if e.User != nil {
				row.name = e.User.DisplayName

				if row.name == "" {
					row.name = e.User.Username
				}
			}

			if e.Favorite != nil && e.Favorite.Name != "" {
				row.extra = "mostly " + e.Favorite.Name
			}

			out[i] = row
		}

		return out, nil
	default:
		rows, err := q.GetTopVotedBots(ctx, voteBoardSize)

		if err != nil {
			return nil, err
		}

		out := make([]voteBoardRow, len(rows))

		var wg sync.WaitGroup

		for i, r := range rows {
			out[i] = voteBoardRow{
				name:  "Unknown bot",
				url:   fmt.Sprintf("%s/bots/%s", frontendURL(), r.BotID),
				votes: int(r.ApproximateVotes),
			}

			wg.Go(func() {
				user, err := dovewing.GetUser(ctx, r.BotID, state.DovewingPlatformDiscord)

				if err == nil && user != nil {
					if user.DisplayName != "" {
						out[i].name = user.DisplayName
					} else if user.Username != "" {
						out[i].name = user.Username
					}
				}
			})
		}

		wg.Wait()
		return out, nil
	}
}

func voteBoardName(s string) string {
	return kbClip(strings.TrimSpace(kbMarkdown.Replace(s)), 40)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}

	return fmt.Sprintf("%d %s", n, many)
}

func voteBoardView(kind string, rows []voteBoardRow) kbView {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## %s\n", voteBoardTitles[kind])

	if len(rows) == 0 {
		sb.WriteString("No votes yet this month. Be the first!\n")
	}

	for i, r := range rows {
		name := voteBoardName(r.name)

		if r.url != "" {
			name = fmt.Sprintf("[%s](<%s>)", name, r.url)
		}

		fmt.Fprintf(&sb, "%s **%s** · %s", medal(i), name, plural(r.votes, "vote", "votes"))

		if r.extra != "" {
			fmt.Fprintf(&sb, " · %s", voteBoardName(r.extra))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("-# Votes reset at the start of each month.")

	button := func(k, label string) discord.InteractiveComponent {
		b := discord.NewSecondaryButton(label, voteBoardPrefix+k)

		if k == kind {
			b = discord.NewPrimaryButton(label, voteBoardPrefix+k).AsDisabled()
		}

		return b
	}

	full := frontendURL() + "/leaderboard"

	if kind != voteBoardVoters {
		full = fmt.Sprintf("%s/%s?sort=votes", frontendURL(), kind)
	}

	return kbView{
		content: sb.String(),
		components: []discord.ContainerComponent{
			discord.NewActionRow(
				button(voteBoardBots, "Bots"),
				button(voteBoardServers, "Servers"),
				button(voteBoardVoters, "Voters"),
				discord.NewLinkButton("See more", full),
			),
		},
	}
}

func cmdVoteLeaderboard() *Command {
	return &Command{
		Name:        "leaderboard",
		Category:    "Votes",
		Description: "This month's most voted bots and servers, and the top voters",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "show",
				Description: "Which leaderboard to show",
				Choices: []discord.ApplicationCommandOptionChoiceString{
					{Name: "Bots", Value: voteBoardBots},
					{Name: "Servers", Value: voteBoardServers},
					{Name: "Voters", Value: voteBoardVoters},
				},
			},
		},
		Run: func(c *Ctx) error {
			kind := c.Option("show", 0)

			if _, ok := voteBoardTitles[kind]; !ok {
				kind = voteBoardBots
			}

			if err := c.Defer(); err != nil {
				return err
			}

			rows, err := loadVoteBoard(c.Context, kind)

			if err != nil {
				return err
			}

			return c.Send(voteBoardView(kind, rows).create())
		},
	}
}

func onVoteBoardComponent(ctx context.Context, e *events.ComponentInteractionCreate) {
	kind := strings.TrimPrefix(e.Data.CustomID(), voteBoardPrefix)

	if _, ok := voteBoardTitles[kind]; !ok {
		return
	}

	rows, err := loadVoteBoard(ctx, kind)

	if err != nil {
		state.Logger.Error("Failed to load a vote leaderboard", zap.String("kind", kind), zap.Error(err))
		_ = e.CreateMessage(discord.MessageCreate{
			Content: "I couldn't load that leaderboard just now. Try again in a moment.",
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	view := voteBoardView(kind, rows)

	if meta := e.Message.InteractionMetadata; meta != nil && meta.User.ID != e.User().ID {
		msg := view.create()
		msg.Flags |= discord.MessageFlagEphemeral
		_ = e.CreateMessage(msg)
		return
	}

	if err := e.UpdateMessage(view.update()); err != nil {
		state.Logger.Error("Failed to update a vote leaderboard", zap.Error(err))
	}
}
