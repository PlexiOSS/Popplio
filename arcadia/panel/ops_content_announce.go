// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
)

func announceBlogPost(entry types.BlogCreateEntry) {
	channelID := state.Config.Channels.BlogAnnouncements

	if channelID == 0 {
		return
	}

	_, err := state.Discord.Rest().CreateMessage(channelID, discord.MessageCreate{
		Content: state.Config.Meta.BlogAnnounceMentions,
		Embeds: []discord.Embed{
			{
				Title:       "New Blog Post: " + entry.Title,
				URL:         state.Config.Sites.Frontend + "/blog/" + entry.Slug,
				Description: entry.Description,
				Color:       0x00ff00,
			},
		},
	})

	if err != nil {
		state.Logger.Warn("Failed to announce new blog post", zap.String("slug", entry.Slug), zap.Error(err))
	}
}

func announceChangelogEntry(entry types.ChangelogCreateEntry) {
	if !entry.Published {
		return
	}

	channelID := state.Config.Channels.ChangelogAnnouncements

	if channelID == 0 {
		return
	}

	fields := make([]discord.EmbedField, 0, 4)

	addField := func(name string, items []string) {
		if len(items) == 0 {
			return
		}

		value := "- " + joinTruncated(items, "\n- ", 1000)

		fields = append(fields, discord.EmbedField{Name: name, Value: value})
	}

	addField("Added", entry.Added)
	addField("Updated", entry.Updated)
	addField("Fixed", entry.Fixed)
	addField("Removed", entry.Removed)

	_, err := state.Discord.Rest().CreateMessage(channelID, discord.MessageCreate{
		Content: state.Config.Meta.ChangelogAnnounceMentions,
		Embeds: []discord.Embed{
			{
				Title:       projectLabel(entry.Project) + " " + entry.Version + " Released",
				URL:         state.Config.Sites.Frontend + "/changelog",
				Description: entry.ExtraDescription,
				Color:       0x00ff00,
				Fields:      fields,
			},
		},
	})

	if err != nil {
		state.Logger.Warn("Failed to announce new changelog entry",
			zap.String("project", entry.Project), zap.String("version", entry.Version), zap.Error(err))
	}
}

func projectLabel(project string) string {
	switch project {
	case "popplio":
		return "Popplio"
	case "omniplex":
		return "Omniplex"
	case "keel":
		return "Keel"
	default:
		return project
	}
}

func joinTruncated(items []string, sep string, max int) string {
	result := ""

	for i, item := range items {
		piece := item

		if i > 0 {
			piece = sep + item
		}

		if len(result)+len(piece) > max {
			return result + "..."
		}

		result += piece
	}

	return result
}
