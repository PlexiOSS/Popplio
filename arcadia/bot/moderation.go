package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	djson "github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
)

// Guild moderation and self-serve support commands for the main server.
//
// The moderation commands (/kick, /ban, /timeout, /warn) all go through
// checkModerationTarget before touching Discord, so staff hierarchy holds here
// the same way it holds in the panel and the RPC layer: nobody can act on a
// staff member ranked at or above themselves. StaffResync is what keeps
// staff_positions in sync with the Discord roles this check reads.

func registerModerationCommands() {
	register(
		cmdKick(),
		cmdBan(),
		cmdTimeout(),
		cmdWarn(),
		cmdKB(),
		cmdTicket(),
		cmdStaffInfo(),
	)
}

// resolveTargetUser reads a user-option target and trims Discord's mention
// formatting off of it, the same way the testing commands read "bot".
func resolveTargetUser(c *Ctx, name string, index int) string {
	return strings.Trim(c.Option(name, index), "<@!>")
}

// checkModerationCall guards every moderation command: caller must be staff,
// in the main server, and hold the given permission.
func checkModerationCall(c *Ctx, perm perms.Perm) error {
	if err := mainServer(c); err != nil {
		return err
	}

	return requirePerm(c, perm)
}

// checkModerationTarget refuses to let a caller act on a target who is
// themselves staff at a rank equal to or more senior than the caller's own.
// A target with no staff_members row ranks below everyone and is never
// blocked here.
func checkModerationTarget(c *Ctx, targetID string) error {
	if targetID == c.Author.ID.String() {
		return fmt.Errorf("you cannot use this on yourself")
	}

	actor, err := perms.LoadStaff(c.Context, c.Author.ID.String())

	if err != nil {
		return err
	}

	target, err := perms.LoadStaff(c.Context, targetID)

	// A target with no staff_members row is not staff, LoadStaff wraps
	// pgx.ErrNoRows for that case and Rank() would come out as NoRank anyway,
	// but it's cheaper to just let the not-staff case fall through as unranked.
	if err != nil {
		return nil
	}

	if target.Rank() <= actor.Rank() {
		return fmt.Errorf("you cannot moderate a staff member at or above your own rank")
	}

	return nil
}

// logModeration posts the mod-log embed every moderation command shares.
func logModeration(action, targetID, reason string, colour int) error {
	return impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       action,
			Description: fmt.Sprintf("Target: <@%s>", targetID),
			Fields:      []discord.EmbedField{{Name: "Reason", Value: reason, Inline: impls.InlineTrue()}},
			Color:       colour,
		}},
	})
}

func cmdKick() *Command {
	return &Command{
		Name:        "kick",
		Category:    "Moderation",
		Description: "Kick a member from the main server",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to kick", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Reason for the kick", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := checkModerationCall(c, perms.StaffModerateGuild); err != nil {
				return err
			}

			targetID := resolveTargetUser(c, "user", 0)
			reason := c.Option("reason", 1)

			if err := checkModerationTarget(c, targetID); err != nil {
				return err
			}

			targetSnowflake, err := parseSnowflake(targetID)

			if err != nil {
				return err
			}

			if err := dclient.Get().Rest().RemoveMember(c.GuildID, targetSnowflake); err != nil {
				return fmt.Errorf("failed to kick: %w", err)
			}

			if err := logModeration("Member Kicked", targetID, reason, impls.ColourRed); err != nil {
				state.Logger.Warn("Failed to post kick to mod log")
			}

			return c.Ok(fmt.Sprintf("Kicked <@%s>.", targetID))
		},
	}
}

func cmdBan() *Command {
	return &Command{
		Name:        "ban",
		Category:    "Moderation",
		Description: "Ban a member from the main server",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to ban", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Reason for the ban", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := checkModerationCall(c, perms.StaffModerateGuild); err != nil {
				return err
			}

			targetID := resolveTargetUser(c, "user", 0)
			reason := c.Option("reason", 1)

			if err := checkModerationTarget(c, targetID); err != nil {
				return err
			}

			targetSnowflake, err := parseSnowflake(targetID)

			if err != nil {
				return err
			}

			if err := dclient.Get().Rest().AddBan(c.GuildID, targetSnowflake, 0, nil); err != nil {
				return fmt.Errorf("failed to ban: %w", err)
			}

			if err := logModeration("Member Banned", targetID, reason, impls.ColourRed); err != nil {
				state.Logger.Warn("Failed to post ban to mod log")
			}

			return c.Ok(fmt.Sprintf("Banned <@%s>.", targetID))
		},
	}
}

func cmdTimeout() *Command {
	return &Command{
		Name:        "timeout",
		Category:    "Moderation",
		Description: "Time out a member in the main server",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to time out", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Reason for the timeout", Required: true},
			discord.ApplicationCommandOptionString{Name: "duration", Description: "e.g. 10m, 1h, 2d (max 28d, clears if omitted)"},
		},
		Run: func(c *Ctx) error {
			if err := checkModerationCall(c, perms.StaffModerateGuild); err != nil {
				return err
			}

			targetID := resolveTargetUser(c, "user", 0)
			reason := c.Option("reason", 1)
			duration := c.Option("duration", 2)

			if err := checkModerationTarget(c, targetID); err != nil {
				return err
			}

			targetSnowflake, err := parseSnowflake(targetID)

			if err != nil {
				return err
			}

			update := discord.MemberUpdate{}

			if strings.TrimSpace(duration) == "" {
				update.CommunicationDisabledUntil = djson.NullPtr[time.Time]()
			} else {
				d, err := parseTimeoutDuration(duration)

				if err != nil {
					return err
				}

				if d > 28*24*time.Hour {
					return fmt.Errorf("timeouts cannot exceed 28 days")
				}

				update.CommunicationDisabledUntil = djson.NewNullablePtr(time.Now().Add(d))
			}

			if _, err := dclient.Get().Rest().UpdateMember(c.GuildID, targetSnowflake, update); err != nil {
				return fmt.Errorf("failed to time out: %w", err)
			}

			if err := logModeration("Member Timed Out", targetID, reason, impls.ColourRed); err != nil {
				state.Logger.Warn("Failed to post timeout to mod log")
			}

			if strings.TrimSpace(duration) == "" {
				return c.Ok(fmt.Sprintf("Cleared the timeout on <@%s>.", targetID))
			}

			return c.Ok(fmt.Sprintf("Timed out <@%s> for %s.", targetID, duration))
		},
	}
}

func cmdWarn() *Command {
	return &Command{
		Name:        "warn",
		Category:    "Moderation",
		Description: "Warn a member",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to warn", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Reason for the warning", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := checkModerationCall(c, perms.StaffWarnUsers); err != nil {
				return err
			}

			targetID := resolveTargetUser(c, "user", 0)
			reason := c.Option("reason", 1)

			if err := checkModerationTarget(c, targetID); err != nil {
				return err
			}

			targetSnowflake, err := parseSnowflake(targetID)

			if err != nil {
				return err
			}

			dm, dmErr := dclient.Get().Rest().CreateDMChannel(targetSnowflake)

			if dmErr == nil {
				// A closed DM is not a reason to fail the warning itself, only to
				// skip telling the target directly - the mod log still records it.
				_, _ = dclient.Get().Rest().CreateMessage(dm.ID(), discord.MessageCreate{
					Embeds: []discord.Embed{{
						Title:       "You have been warned",
						Description: reason,
						Color:       impls.ColourRed,
					}},
				})
			}

			if err := logModeration("Member Warned", targetID, reason, impls.ColourRed); err != nil {
				state.Logger.Warn("Failed to post warning to mod log")
			}

			return c.Ok(fmt.Sprintf("Warned <@%s>.", targetID))
		},
	}
}

// cmdKB points a user at the Knowledge Base without staff retyping the link
// every time someone asks the same question.
func cmdKB() *Command {
	return &Command{
		Name:        "kb",
		Category:    "Support",
		Description: "Link the Knowledge Base",
		Run: func(c *Ctx) error {
			return c.Say(fmt.Sprintf("Check out our Knowledge Base: %s/kb", state.Config.Sites.Frontend.Parse()))
		},
	}
}

// cmdTicket points a user at how to open a support ticket in-site.
func cmdTicket() *Command {
	return &Command{
		Name:        "ticket",
		Category:    "Support",
		Description: "Explain how to open a support ticket",
		Run: func(c *Ctx) error {
			return c.Say(fmt.Sprintf(
				"Need direct help? Open a support ticket at %s/tickets and a staff member will get to it.",
				state.Config.Sites.Frontend.Parse()))
		},
	}
}

// cmdStaffInfo explains the staff hierarchy the moderation commands enforce,
// so "why couldn't I moderate them" has a self-serve answer.
func cmdStaffInfo() *Command {
	return &Command{
		Name:        "staffinfo",
		Category:    "Support",
		Description: "Explain the staff hierarchy",
		Run: func(c *Ctx) error {
			return c.Say("Staff moderation commands follow the staff role hierarchy: " +
				"you cannot kick, ban, time out or warn a staff member ranked at or above your own role. " +
				"Roles and their rank are kept in sync with this server's staff roles.")
		},
	}
}

// parseTimeoutDuration parses a duration written with a day suffix in
// addition to what time.ParseDuration already understands, since Discord
// timeouts commonly run in days and Go's stdlib has no unit for one.
func parseTimeoutDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	if strings.HasSuffix(s, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)

		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}

		return time.Duration(days * float64(24*time.Hour)), nil
	}

	d, err := time.ParseDuration(s)

	if err != nil {
		return 0, fmt.Errorf("invalid duration %q, try e.g. 10m, 1h, 2d", s)
	}

	return d, nil
}

func parseSnowflake(id string) (snowflake.ID, error) {
	return snowflake.Parse(id)
}
