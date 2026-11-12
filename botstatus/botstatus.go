// Copyright (C) 2026 NodeByte LTD

package botstatus

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"go.uber.org/zap"
)

const Interval = 5 * time.Minute

var siteHost = state.SiteHost

type Kind int

const (
	Watching Kind = iota
	Listening
	Competing
	Custom
)

type Line struct {
	Kind Kind
	Text string
}

func (l Line) presence() gateway.PresenceOpt {
	switch l.Kind {
	case Listening:
		return gateway.WithListeningActivity(l.Text)
	case Competing:
		return gateway.WithCompetingActivity(l.Text)
	case Custom:
		return gateway.WithCustomActivity(l.Text)
	default:
		return gateway.WithWatchingActivity(l.Text)
	}
}

type Counts = db.GetBotStatusCountsRow

func Omi(c Counts) []Line {
	return []Line{
		{Watching, fmt.Sprintf("%s bots and %s servers", Number(c.ListedBots), Number(c.ListedServers))},
		{Listening, "/help"},
		{Custom, "Find your next bot at " + siteHost()},
		{Competing, "this month's vote leaderboard"},
	}
}

func Arcadia(c Counts) []Line {
	waiting := c.PendingBots + c.PendingServers
	queue := Line{Watching, "an empty review queue"}

	if waiting > 0 {
		queue = Line{Custom, fmt.Sprintf("Reviewing %s in the queue", plural(waiting, "listing"))}
	}

	return []Line{queue, {Listening, "/queue"}}
}

func Servers(c Counts) []Line {
	return []Line{
		{Watching, plural(c.ListedServers, "listed server")},
		{Listening, "/vote and /bump"},
		{Custom, "List your server at " + siteHost() + "/servers/add"},
	}
}

func Run(ctx context.Context, name string, client func() bot.Client, lines func(Counts) []Line) {
	go func() {
		next := 0
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}

			if c := client(); c != nil {
				counts, err := db.New(state.Pool).GetBotStatusCounts(ctx)

				if err != nil {
					state.Logger.Warn("botstatus: failed to load counts", zap.String("bot", name), zap.Error(err))
				} else if all := lines(counts); len(all) > 0 {
					line := all[next%len(all)]
					next++

					if err := apply(ctx, c, line); err != nil {
						state.Logger.Debug("botstatus: failed to set presence", zap.String("bot", name), zap.Error(err))
					}
				}
			}

			timer.Reset(Interval)
		}
	}()
}

func apply(ctx context.Context, c bot.Client, line Line) error {
	opts := []gateway.PresenceOpt{gateway.WithOnlineStatus(discord.OnlineStatusOnline), line.presence()}

	if c.HasShardManager() {
		var firstErr error

		for id := range c.ShardManager().Shards() {
			if err := c.SetPresenceForShard(ctx, id, opts...); err != nil && firstErr == nil {
				firstErr = err
			}
		}

		return firstErr
	}

	return c.SetPresence(ctx, opts...)
}

func plural(n int64, word string) string {
	if n == 1 {
		return "1 " + word
	}

	return Number(n) + " " + word + "s"
}

func Number(n int64) string {
	s := strconv.FormatInt(n, 10)

	if n < 0 {
		return "-" + Number(-n)
	}

	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}

	return s
}
