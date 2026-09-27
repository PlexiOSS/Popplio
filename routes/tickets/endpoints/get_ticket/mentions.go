package get_ticket

import (
	"context"
	"regexp"

	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/disgoorg/snowflake/v2"
)

const maxResolvedUsers = 50

var (
	userMention    = regexp.MustCompile(`<@!?(\d{15,21})>`)
	roleMention    = regexp.MustCompile(`<@&(\d{15,21})>`)
	channelMention = regexp.MustCompile(`<#(\d{15,21})>`)
)

func messageTexts(m types.Message) []string {
	texts := []string{m.Content}
	for _, e := range m.Embeds {
		texts = append(texts, e.Title, e.Description)
		for _, f := range e.Fields {
			texts = append(texts, f.Name, f.Value)
		}
		if e.Footer != nil {
			texts = append(texts, e.Footer.Text)
		}
	}
	return texts
}

func collect(re *regexp.Regexp, texts []string, into map[string]struct{}) {
	for _, t := range texts {
		for _, m := range re.FindAllStringSubmatch(t, -1) {
			into[m[1]] = struct{}{}
		}
	}
}

func resolveMentions(ctx context.Context, ticket *types.Ticket) *types.TicketMentions {
	var texts []string
	known := map[string]struct{}{ticket.UserID: {}}
	for _, m := range ticket.Messages {
		texts = append(texts, messageTexts(m)...)
		known[m.AuthorID] = struct{}{}
	}

	users, roles, channels := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	collect(userMention, texts, users)
	collect(roleMention, texts, roles)
	collect(channelMention, texts, channels)

	out := &types.TicketMentions{
		Users:    map[string]*dovetypes.PlatformUser{},
		Roles:    map[string]types.TicketMentionRole{},
		Channels: map[string]string{},
	}

	resolved := 0
	for id := range users {
		if _, ok := known[id]; ok || resolved >= maxResolvedUsers {
			continue
		}
		resolved++
		if u, err := dovewing.GetUser(ctx, id, state.DovewingPlatformDiscord); err == nil && u != nil {
			out.Users[id] = u
		}
	}

	if state.Discord == nil {
		return out
	}

	caches := state.Discord.Caches()
	guilds := []snowflake.ID{state.Config.Servers.Main, state.Config.Servers.Staff, state.Config.Servers.Testing}

	for id := range roles {
		roleID, err := snowflake.Parse(id)
		if err != nil {
			continue
		}
		for _, g := range guilds {
			if role, ok := caches.Role(g, roleID); ok {
				out.Roles[id] = types.TicketMentionRole{Name: role.Name, Color: role.Color}
				break
			}
		}
	}

	for id := range channels {
		channelID, err := snowflake.Parse(id)
		if err != nil {
			continue
		}
		if ch, ok := caches.Channel(channelID); ok {
			out.Channels[id] = ch.Name()
		}
	}

	return out
}
