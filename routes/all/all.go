// Copyright (C) 2026 NodeByte LTD

package all

import (
	"popplio/types"
	poplhooks "popplio/webhooks"

	"popplio/routes/alerts"
	"popplio/routes/apps"
	"popplio/routes/auth"
	"popplio/routes/badges"
	"popplio/routes/blogs"
	"popplio/routes/bots"
	"popplio/routes/changelogs"
	"popplio/routes/diagnostics"
	"popplio/routes/emojis"
	"popplio/routes/health"
	linkedrolesroutes "popplio/routes/linkedroles"
	"popplio/routes/list"
	notifrouter "popplio/routes/notifications"
	"popplio/routes/packs"
	"popplio/routes/payments"
	"popplio/routes/platform"
	"popplio/routes/reminders"
	reportsrouter "popplio/routes/reports"
	"popplio/routes/reviews"
	"popplio/routes/servers"
	"popplio/routes/servertemplates"
	"popplio/routes/shop"
	"popplio/routes/sounds"
	"popplio/routes/staff"
	"popplio/routes/stickers"
	"popplio/routes/tasks"
	"popplio/routes/teams"
	"popplio/routes/themes"
	"popplio/routes/tickets"
	"popplio/routes/users"
	"popplio/routes/vanity"
	"popplio/routes/votes"
	"popplio/routes/webhooks"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const Version = "1.23.0"

func Routers() []uapi.APIRouter {
	return []uapi.APIRouter{
		alerts.Router{},
		apps.Router{},
		auth.Router{},
		badges.Router{},
		linkedrolesroutes.Router{},
		blogs.Router{},
		bots.Router{},
		changelogs.Router{},
		diagnostics.Router{},
		emojis.Router{},
		health.Router{},
		list.Router{},
		notifrouter.Router{},
		packs.Router{},
		payments.Router{},
		platform.Router{},
		reminders.Router{},
		reportsrouter.Router{},
		reviews.Router{},
		servers.Router{},
		servertemplates.Router{},
		shop.Router{},
		sounds.Router{},
		staff.Router{},
		stickers.Router{},
		tasks.Router{},
		teams.Router{},
		themes.Router{},
		tickets.Router{},
		users.Router{},
		vanity.Router{},
		votes.Router{},
		webhooks.Router{},
	}
}

func SetupDocs(apiURL string) {
	docs.DocsSetupData = &docs.SetupData{
		URL:               apiURL,
		ServerDescription: "Popplio",
		ErrorStruct:       types.ApiError{},
		Info: docs.Info{
			Title:          "Omniplex API",
			TermsOfService: "https://omniplex.gg/legal/terms",
			Version:        Version,
			Description:    "RESTful API and Backend Services for Omniplex",
			Contact: docs.Contact{
				Name: "Omniplex Support",
				URL:  "https://discord.gg/KBCRuBKrHe",
			},
			License: docs.License{
				Name: "AGPL-3.0",
				URL:  "https://opensource.org/licenses/AGPL-3.0",
			},
		},
	}

	docs.Setup()
	poplhooks.RegisterDocs()

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Should be prefixed with `User ` in `Authorization` header.")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Should be prefixed with `Bot ` in `Authorization` header.")
	docs.AddSecuritySchema("server", "Server-Auth", "Requires a server API session token. Should be prefixed with `Server ` in `Authorization` header.")
	docs.AddSecuritySchema("team", "Team-Auth", "Requires a team API session token. Should be prefixed with `Team ` in `Authorization` header.")
}
