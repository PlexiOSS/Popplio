package bot

import (
	"fmt"
	"strings"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

func cmdLock() *Command {
	return &Command{
		Name:        "lock",
		Category:    "Moderation",
		Description: "Deny @everyone's Send Messages in this channel (or a chosen one)",
		Checks:      []Check{isStaff, mainOrTestingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Why the channel is being locked", Required: true},
			discord.ApplicationCommandOptionChannel{Name: "channel", Description: "Channel to lock (defaults to this one)"},
		},
		Run: func(c *Ctx) error {
			target, err := resolveLockdownChannel(c, 1)

			if err != nil {
				return err
			}

			return runLockdown(c, true, target, c.Option("reason", 0))
		},
	}
}

func cmdUnlock() *Command {
	return &Command{
		Name:        "unlock",
		Category:    "Moderation",
		Description: "Restore @everyone's Send Messages in this channel (or a chosen one)",
		Checks:      []Check{isStaff, mainOrTestingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionChannel{Name: "channel", Description: "Channel to unlock (defaults to this one)"},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Why the channel is being unlocked"},
		},
		Run: func(c *Ctx) error {
			target, err := resolveLockdownChannel(c, 0)

			if err != nil {
				return err
			}

			return runLockdown(c, false, target, c.Option("reason", 1))
		},
	}
}

// resolveLockdownChannel reads the optional "channel" option, defaulting to
// the invoking channel.
func resolveLockdownChannel(c *Ctx, index int) (snowflake.ID, error) {
	raw := c.Option("channel", index)

	if raw == "" {
		return c.ChannelID, nil
	}

	id, err := snowflake.Parse(strings.Trim(raw, "<#>"))

	if err != nil {
		return 0, fmt.Errorf("%q is not a valid channel", raw)
	}

	return id, nil
}

func runLockdown(c *Ctx, lock bool, target snowflake.ID, reason string) error {
	if err := requirePerm(c, perms.StaffLockChannels); err != nil {
		return err
	}

	action, verb := "unlock", "Channel Unlocked"

	if lock {
		action, verb = "lock", "Channel Locked"

		if reason == "" {
			return fmt.Errorf("a reason is required to lock a channel")
		}
	} else if reason == "" {
		reason = "Channel unlocked"
	}

	var err error

	if lock {
		err = impls.LockChannel(c.GuildID, target, reason)
	} else {
		err = impls.UnlockChannel(c.GuildID, target, reason)
	}

	if err != nil {
		return err
	}

	if err := writeModCase(c.Context, modCase{GuildID: c.GuildID, ModeratorID: c.Author.ID, Action: action, Reason: fmt.Sprintf("<#%s>: %s", target, reason)}); err != nil {
		state.Logger.Error("Failed to record lockdown mod case", zap.Error(err))
	}

	if err := logModeration(c, verb, 0, fmt.Sprintf("<#%s>: %s", target, reason)); err != nil {
		state.Logger.Error("Failed to log lockdown", zap.Error(err))
	}

	if lock {
		return c.Ok(fmt.Sprintf("Locked <#%s>. This only denies @everyone — a role with its own Send Messages allow-override on this channel will not be silenced by it.", target))
	}

	return c.Ok(fmt.Sprintf("Unlocked <#%s>.", target))
}
