package bot

import (
	"fmt"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

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

			rows, err := state.Pool.Query(c.Context,
				`SELECT action, moderator_id, reason, created_at FROM mod_cases
				 WHERE guild_id = $1 AND user_id = $2
				 ORDER BY created_at DESC LIMIT 10`,
				c.GuildID.String(), target.String())

			if err != nil {
				return err
			}

			defer rows.Close()

			var fields []discord.EmbedField

			for rows.Next() {
				var action, moderatorID, reason string
				var createdAt time.Time

				if err := rows.Scan(&action, &moderatorID, &reason, &createdAt); err != nil {
					return err
				}

				fields = append(fields, discord.EmbedField{
					Name:  fmt.Sprintf("%s — <t:%d:f>", strings.ToUpper(action[:1])+action[1:], createdAt.Unix()),
					Value: fmt.Sprintf("By <@%s>: %s", moderatorID, reason),
				})
			}

			if err := rows.Err(); err != nil {
				return err
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
