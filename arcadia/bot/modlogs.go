package bot

import (
	"fmt"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

type ModCaseRow struct {
	GuildID     string    `db:"guild_id"`
	UserID      string    `db:"user_id"`
	ModeratorID string    `db:"moderator_id"`
	Action      string    `db:"action"`
	Reason      string    `db:"reason"`
	CreatedAt   time.Time `db:"created_at"`
}

func cmdModlogs() *Command {
	return &Command{
		Name:        "modlogs",
		Category:    "Moderation",
		Description: "Show a member's recent moderation history",
		Checks:      []Check{isStaff, mainOrTestingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "user", Description: "The member to look up", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := requirePerm(c, perms.StaffViewModCases); err != nil {
				return err
			}

			target, err := resolveTargetUser(c, "user", 0)

			if err != nil {
				return err
			}

			rows, err := db.New(state.Pool).GetModCasesForUser(c.Context, db.GetModCasesForUserParams{
				GuildID: c.GuildID.String(),
				UserID:  target.String(),
			})

			if err != nil {
				return err
			}

			var fields []discord.EmbedField

			for _, row := range rows {
				action, moderatorID, reason, createdAt := row.Action, row.ModeratorID, row.Reason, row.CreatedAt.Time

				fields = append(fields, discord.EmbedField{
					Name:  fmt.Sprintf("%s — <t:%d:f>", strings.ToUpper(action[:1])+action[1:], createdAt.Unix()),
					Value: fmt.Sprintf("By <@%s>: %s", moderatorID, reason),
				})
			}

			if len(fields) == 0 {
				return c.Ok(fmt.Sprintf("<@%s> has no recorded moderation history in this server.", target))
			}

			return c.Send(discord.MessageCreate{Embeds: []discord.Embed{{
				Title:  fmt.Sprintf("Moderation history for %s", target),
				Fields: fields,
				Color:  impls.ColourBlurple,
				Footer: impls.Footer("Showing up to the 10 most recent cases"),
			}}})
		},
	}
}
