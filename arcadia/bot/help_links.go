package bot

import (
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// Self-serve commands anyone (not just staff) can use to point themselves at
// the Knowledge Base, ticket support, and the staff hierarchy pages, rather
// than staff retyping the same links by hand in chat every time.

func registerHelpLinkCommands() {
	register(cmdKb(), cmdTicket(), cmdStaffInfo())
}

// frontendURL is always the production site, even from a dev/staging
// instance of this bot the staff server it answers in is shared across
// environments, and a link should always send someone somewhere real.
func frontendURL() string {
	return state.Config.Sites.Frontend
}

func cmdKb() *Command {
	return &Command{
		Name:        "kb",
		Category:    "Help",
		Description: "Get a link to the Knowledge Base",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "topic", Description: "What you're looking for help with"},
		},
		Run: func(c *Ctx) error {
			topic := c.Option("topic", 0)

			if topic == "" {
				return c.Say(fmt.Sprintf("Check out the Knowledge Base: %s/kb", frontendURL()))
			}

			return c.Say(fmt.Sprintf("Try searching for **%s** on the Knowledge Base: %s/kb", topic, frontendURL()))
		},
	}
}

func cmdTicket() *Command {
	return &Command{
		Name:        "ticket",
		Category:    "Help",
		Description: "How to open a support ticket",
		Run: func(c *Ctx) error {
			return c.Say(fmt.Sprintf(
				"Need help from a human? Open a support ticket at %s/tickets/new staff can see and reply to it from there, and you can track its status from your dashboard's Tickets tab.",
				frontendURL(),
			))
		},
	}
}

func cmdStaffInfo() *Command {
	return &Command{
		Name:        "staffinfo",
		Category:    "Help",
		Description: "How the staff team is structured and what each position can do",
		Run: func(c *Ctx) error {
			return c.Say(fmt.Sprintf("Staff hierarchy, permissions, and how reports/tickets get handled: %s/staff", frontendURL()))
		},
	}
}
