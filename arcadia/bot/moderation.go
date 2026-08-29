package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Guild moderation: kick, ban, timeout, and a log-only warn.
//
// Every one of these refuses to act on a staff member who ranks at or above
// the caller's own rank. staff_positions IS the Discord role hierarchy (the
// resync task keeps staff_members.positions in step with staff-server role
// assignments — see staffresync.go), so comparing StaffGrants.Rank() is
// simultaneously the Discord check and the Omniplex check: there is only one
// hierarchy here, not two competing ones. This mirrors canEditRole's "at or
// above the caller's own rank" rule in staffmgmt.go, applied to people
// instead of positions.

const maxTimeout = 28 * 24 * time.Hour

func registerModerationCommands() {
	register(cmdKick(), cmdBan(), cmdTimeout(), cmdWarn(), cmdPurge(), cmdLock(), cmdUnlock(), cmdModlogs())
}

// modCase is one row written to mod_cases: a durable, queryable moderation
// history behind the Discord-only embeds logModeration posts.
type modCase struct {
	GuildID     snowflake.ID
	UserID      snowflake.ID
	ModeratorID snowflake.ID
	Action      string
	Reason      string
}

// writeModCase persists a moderation action. Best-effort, same contract as
// logModeration: called after the action already succeeded, a failure here
// is logged by the caller but never undoes it.
func writeModCase(ctx context.Context, mc modCase) error {
	return db.New(state.Pool).InsertModCase(ctx, db.InsertModCaseParams{
		GuildID:     mc.GuildID.String(),
		UserID:      mc.UserID.String(),
		ModeratorID: mc.ModeratorID.String(),
		Action:      mc.Action,
		Reason:      mc.Reason,
	})
}

// logPurge posts a mod-log entry for a purge. Separate from logModeration
// because a purge may have no single target user (an unfiltered purge), and
// logModeration's embed always renders a "User" field.
func logPurge(c *Ctx, channelID snowflake.ID, count int, filterUser snowflake.ID) error {
	fields := []discord.EmbedField{
		{Name: "Channel", Value: fmt.Sprintf("<#%s>", channelID), Inline: impls.InlineTrue()},
		{Name: "Count", Value: fmt.Sprintf("%d", count), Inline: impls.InlineTrue()},
		{Name: "Staff", Value: fmt.Sprintf("<@%s>", c.Author.ID), Inline: impls.InlineTrue()},
	}

	if filterUser != 0 {
		fields = append(fields, discord.EmbedField{Name: "Filtered To", Value: fmt.Sprintf("<@%s>", filterUser), Inline: impls.InlineTrue()})
	}

	return impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:  "Messages Purged",
			Fields: fields,
			Color:  impls.ColourRed,
		}},
	})
}

// resolveTargetUser parses a mention or raw id option into a snowflake.
func resolveTargetUser(c *Ctx, optName string, index int) (snowflake.ID, error) {
	raw := strings.Trim(c.Option(optName, index), "<@!>")

	if raw == "" {
		return 0, errors.New("a user is required")
	}

	id, err := snowflake.Parse(raw)

	if err != nil {
		return 0, fmt.Errorf("%q is not a valid user", raw)
	}

	return id, nil
}

// checkModerationCall runs every guard a moderation command needs: the guild
// itself, the staff permission, not targeting yourself, and rank.
func checkModerationCall(c *Ctx, perm perms.Perm, target snowflake.ID) error {
	if c.GuildID == 0 {
		return errors.New("this command can only be used in a server")
	}

	if target == c.Author.ID {
		return errors.New("you can't target yourself")
	}

	if err := requirePerm(c, perm); err != nil {
		return err
	}

	grants, err := perms.LoadStaff(c.Context, c.Author.ID.String())

	if err != nil {
		return err
	}

	return checkModerationTarget(c.Context, grants.Rank(), target.String())
}

// checkModerationTarget refuses moderating a staff member who ranks at or
// above the caller. A target with no staff_members row at all (an ordinary
// user) is never restricted here.
func checkModerationTarget(ctx context.Context, actorRank int32, targetUserID string) error {
	target, err := perms.LoadStaff(ctx, targetUserID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return err
	}

	targetRank := target.Rank()

	if targetRank <= actorRank {
		return fmt.Errorf("<@%s> is staff at rank #%d, which is at or above your own rank (#%d) you can't moderate them", targetUserID, targetRank, actorRank)
	}

	return nil
}

// logModeration posts a mod-log entry. Best-effort: a failure here is logged
// by the caller's normal error path but never undoes the action already
// taken, since the moderation action itself already succeeded by the time
// this runs.
func logModeration(c *Ctx, action string, target snowflake.ID, reason string) error {
	return impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: action,
			Fields: []discord.EmbedField{
				{Name: "User", Value: fmt.Sprintf("<@%s> (%s)", target, target), Inline: impls.InlineTrue()},
				{Name: "Staff", Value: fmt.Sprintf("<@%s>", c.Author.ID), Inline: impls.InlineTrue()},
				{Name: "Reason", Value: reason, Inline: impls.InlineFalse()},
			},
			Color: impls.ColourRed,
		}},
	})
}

func cmdKick() *Command {
	return &Command{
		Name:        "kick",
		Category:    "Moderation",
		Description: "Kick a member from this server",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to kick", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Why they're being kicked", Required: true},
		},
		Run: func(c *Ctx) error {
			target, err := resolveTargetUser(c, "user", 0)

			if err != nil {
				return err
			}

			reason := c.Option("reason", 1)

			if err := checkModerationCall(c, perms.StaffModerateGuild, target); err != nil {
				return err
			}

			if err := impls.KickMember(c.GuildID, target, reason); err != nil {
				return err
			}

			if err := writeModCase(c.Context, modCase{GuildID: c.GuildID, UserID: target, ModeratorID: c.Author.ID, Action: "kick", Reason: reason}); err != nil {
				state.Logger.Error("Failed to record kick mod case", zap.Error(err))
			}

			if err := logModeration(c, "Member Kicked", target, reason); err != nil {
				return err
			}

			return c.Ok(fmt.Sprintf("Kicked <@%s>.", target))
		},
	}
}

func cmdBan() *Command {
	return &Command{
		Name:        "ban",
		Category:    "Moderation",
		Description: "Ban a member from this server",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to ban", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Why they're being banned", Required: true},
			discord.ApplicationCommandOptionInt{Name: "delete_days", Description: "Days of their recent messages to delete (0-7)", MinValue: intPtr(0), MaxValue: intPtr(7)},
		},
		Run: func(c *Ctx) error {
			target, err := resolveTargetUser(c, "user", 0)

			if err != nil {
				return err
			}

			reason := c.Option("reason", 1)

			deleteDays := 0

			if raw := c.Option("delete_days", 2); raw != "" {
				deleteDays, err = strconv.Atoi(raw)

				if err != nil || deleteDays < 0 || deleteDays > 7 {
					return errors.New("delete_days must be between 0 and 7")
				}
			}

			if err := checkModerationCall(c, perms.StaffModerateGuild, target); err != nil {
				return err
			}

			if err := impls.BanMember(c.GuildID, target, time.Duration(deleteDays)*24*time.Hour, reason); err != nil {
				return err
			}

			if err := writeModCase(c.Context, modCase{GuildID: c.GuildID, UserID: target, ModeratorID: c.Author.ID, Action: "ban", Reason: reason}); err != nil {
				state.Logger.Error("Failed to record ban mod case", zap.Error(err))
			}

			if err := logModeration(c, "Member Banned", target, reason); err != nil {
				return err
			}

			return c.Ok(fmt.Sprintf("Banned <@%s>.", target))
		},
	}
}

func cmdTimeout() *Command {
	return &Command{
		Name:        "timeout",
		Category:    "Moderation",
		Description: "Time out a member so they can't send messages or join voice",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to time out", Required: true},
			discord.ApplicationCommandOptionString{Name: "duration", Description: "How long, e.g. 10m, 1h, 3d (max 28d)", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Why they're being timed out", Required: true},
		},
		Run: func(c *Ctx) error {
			target, err := resolveTargetUser(c, "user", 0)

			if err != nil {
				return err
			}

			duration, err := parseTimeoutDuration(c.Option("duration", 1))

			if err != nil {
				return err
			}

			reason := c.Option("reason", 2)

			if err := checkModerationCall(c, perms.StaffModerateGuild, target); err != nil {
				return err
			}

			until := time.Now().Add(duration)

			if err := impls.TimeoutMember(c.GuildID, target, until, reason); err != nil {
				return err
			}

			if err := writeModCase(c.Context, modCase{GuildID: c.GuildID, UserID: target, ModeratorID: c.Author.ID, Action: "timeout", Reason: fmt.Sprintf("%s (until <t:%d:f>)", reason, until.Unix())}); err != nil {
				state.Logger.Error("Failed to record timeout mod case", zap.Error(err))
			}

			if err := logModeration(c, "Member Timed Out", target, fmt.Sprintf("%s (until <t:%d:f>)", reason, until.Unix())); err != nil {
				return err
			}

			return c.Ok(fmt.Sprintf("Timed out <@%s> until <t:%d:f>.", target, until.Unix()))
		},
	}
}

func cmdWarn() *Command {
	return &Command{
		Name:        "warn",
		Category:    "Moderation",
		Description: "Send a formal warning to a member (no other action taken)",
		Checks:      []Check{isStaff},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to warn", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "What they're being warned for", Required: true},
		},
		Run: func(c *Ctx) error {
			target, err := resolveTargetUser(c, "user", 0)

			if err != nil {
				return err
			}

			reason := c.Option("reason", 1)

			if err := checkModerationCall(c, perms.StaffWarnUsers, target); err != nil {
				return err
			}

			// Best-effort: a member with DMs closed still gets warned and logged,
			// they just don't hear about it directly.
			_ = impls.SendDM(target, discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title:       "You've received a formal warning",
					Description: reason,
					Color:       impls.ColourRed,
				}},
			})

			if err := writeModCase(c.Context, modCase{GuildID: c.GuildID, UserID: target, ModeratorID: c.Author.ID, Action: "warn", Reason: reason}); err != nil {
				state.Logger.Error("Failed to record warn mod case", zap.Error(err))
			}

			if err := logModeration(c, "Member Warned", target, reason); err != nil {
				return err
			}

			return c.Ok(fmt.Sprintf("Warned <@%s>.", target))
		},
	}
}

func intPtr(i int) *int { return &i }

// parseTimeoutDuration parses a duration string, adding support for a "d"
// (day) suffix time.ParseDuration has no equivalent of, since that's the
// natural unit for a timeout.
func parseTimeoutDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))

	var d time.Duration

	if strings.HasSuffix(raw, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(raw, "d"), 64)

		if err != nil {
			return 0, fmt.Errorf("%q is not a valid duration try e.g. 10m, 1h, 3d", raw)
		}

		d = time.Duration(days * float64(24*time.Hour))
	} else {
		parsed, err := time.ParseDuration(raw)

		if err != nil {
			return 0, fmt.Errorf("%q is not a valid duration try e.g. 10m, 1h, 3d", raw)
		}

		d = parsed
	}

	if d <= 0 {
		return 0, errors.New("duration must be positive")
	}

	if d > maxTimeout {
		return 0, errors.New("duration can't be more than 28 days")
	}

	return d, nil
}
