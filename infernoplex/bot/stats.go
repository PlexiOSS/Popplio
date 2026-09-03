// Copyright (C) 2026 NodeByte LTD

package bot

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/PlexiOSS/Keel/ptr"

	"github.com/disgoorg/disgo/discord"
)

func cmdStats() *Command {
	return &Command{
		Name:        "stats",
		Description: "Bot statistics",
		Run: func(c *Ctx) error {
			var (
				revision = "unknown"
				modified = "unknown"
				version  = "unknown"
			)

			if bi, ok := debug.ReadBuildInfo(); ok {
				version = bi.Main.Version

				for _, setting := range bi.Settings {
					switch setting.Key {
					case "vcs.revision":
						revision = setting.Value
					case "vcs.modified":
						modified = setting.Value
					}
				}
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title: "Infernoplex Statistics",
					Fields: []discord.EmbedField{
						{Name: "Bot Version:", Value: version, Inline: ptr.TruePtr},
						{Name: "Go Version:", Value: runtime.Version(), Inline: ptr.TruePtr},
						{Name: "Git Commit:", Value: revision, Inline: ptr.TruePtr},
						{Name: "Modified:", Value: modified, Inline: ptr.TruePtr},
						{Name: "Built On:", Value: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), Inline: ptr.TruePtr},
					},
				}},
			})
		},
	}
}
