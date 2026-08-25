package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

func cmdPurge() *Command {
	return &Command{
		Name:        "purge",
		Category:    "Moderation",
		Description: "Bulk-delete recent messages in this channel",
		Checks:      []Check{isStaff, mainOrTestingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{Name: "count", Description: "How many messages to delete (1-100)", Required: true, MinValue: intPtr(1), MaxValue: intPtr(100)},
			discord.ApplicationCommandOptionUser{Name: "user", Description: "Only delete this user's messages"},
		},
		Run: func(c *Ctx) error {
			if err := requirePerm(c, perms.StaffPurgeMessages); err != nil {
				return err
			}

			count, err := strconv.Atoi(c.Option("count", 0))

			if err != nil || count < 1 || count > 100 {
				return errors.New("count must be between 1 and 100")
			}

			var filterUser snowflake.ID

			if raw := c.Option("user", 1); raw != "" {
				filterUser, err = snowflake.Parse(strings.Trim(raw, "<@!>"))

				if err != nil {
					return fmt.Errorf("%q is not a valid user", raw)
				}
			}

			if err := c.Defer(); err != nil {
				return err
			}

			// Over-fetch when filtering by user since not every recent
			// message matches; still capped so this can't scan the whole
			// channel's history.
			fetchLimit := count

			if filterUser != 0 {
				fetchLimit = 100
			}

			msgs, err := impls.GetRecentMessages(c.ChannelID, fetchLimit)

			if err != nil {
				return err
			}

			cutoff := time.Now().Add(-14 * 24 * time.Hour)
			ids := make([]snowflake.ID, 0, count)

			for _, m := range msgs {
				if filterUser != 0 && m.Author.ID != filterUser {
					continue
				}

				if m.ID.Time().Before(cutoff) {
					continue
				}

				ids = append(ids, m.ID)

				if len(ids) == count {
					break
				}
			}

			if len(ids) == 0 {
				return c.Fail("No matching messages found (or they're all older than 14 days).")
			}

			reason := fmt.Sprintf("Purge by %s", c.Author.Username)

			if len(ids) == 1 {
				// The bulk-delete endpoint 400s on a single id.
				if err := impls.DeleteMessage(c.ChannelID, ids[0], reason); err != nil {
					return err
				}
			} else if err := impls.PurgeMessages(c.ChannelID, ids, reason); err != nil {
				return err
			}

			if err := writeModCase(c.Context, modCase{GuildID: c.GuildID, UserID: filterUser, ModeratorID: c.Author.ID, Action: "purge", Reason: fmt.Sprintf("%d message(s) in <#%s>", len(ids), c.ChannelID)}); err != nil {
				state.Logger.Error("Failed to record purge mod case", zap.Error(err))
			}

			if err := logPurge(c, c.ChannelID, len(ids), filterUser); err != nil {
				state.Logger.Error("Failed to log purge", zap.Error(err))
			}

			return c.Ok(fmt.Sprintf("Deleted %d message(s).", len(ids)))
		},
	}
}
