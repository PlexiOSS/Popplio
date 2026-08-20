package tasks

import (
	"context"

	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type syncedEmoji struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Animated bool   `json:"animated"`
	URL      string `json:"url"`
}

type syncedSticker struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Format string `json:"format"`
	URL    string `json:"url"`
}

func ServerSync(ctx context.Context) error {
	if err := syncServerMeta(ctx); err != nil {
		return err
	}

	rows, err := state.Pool.Query(ctx, "SELECT server_id FROM servers WHERE show_emojis = true")

	if err != nil {
		return err
	}

	var serverIDs []string

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}

		serverIDs = append(serverIDs, id)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	for _, serverID := range serverIDs {
		guildID, err := snowflake.Parse(serverID)

		if err != nil {
			continue
		}

		emojis, err := dclient.Get().Rest().GetEmojis(guildID)

		if err != nil {
			continue
		}

		stickers, err := dclient.Get().Rest().GetStickers(guildID)

		if err != nil {
			continue
		}

		syncedEmojis := make([]syncedEmoji, 0, len(emojis))

		for _, e := range emojis {
			url := e.URL()
			if e.Animated {
				url = e.URL(discord.WithFormat(discord.FileFormatGIF))
			}

			syncedEmojis = append(syncedEmojis, syncedEmoji{
				ID:       e.ID.String(),
				Name:     e.Name,
				Animated: e.Animated,
				URL:      url,
			})
		}

		syncedStickers := make([]syncedSticker, 0, len(stickers))

		for _, s := range stickers {
			syncedStickers = append(syncedStickers, syncedSticker{
				ID:     s.ID.String(),
				Name:   s.Name,
				Format: stickerFormatName(s.FormatType),
				URL:    s.URL(),
			})
		}

		if _, err := state.Pool.Exec(ctx,
			"UPDATE servers SET emojis = $2, stickers = $3, emojis_synced_at = NOW() WHERE server_id = $1",
			serverID, syncedEmojis, syncedStickers,
		); err != nil {
			return err
		}
	}

	return nil
}

func stickerFormatName(t discord.StickerFormatType) string {
	switch t {
	case discord.StickerFormatTypePNG:
		return "png"
	case discord.StickerFormatTypeAPNG:
		return "apng"
	case discord.StickerFormatTypeLottie:
		return "lottie"
	case discord.StickerFormatTypeGIF:
		return "gif"
	default:
		return "unknown"
	}
}

type syncTarget struct {
	serverID         string
	statsSelfManaged bool
}

func syncServerMeta(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, "SELECT server_id, stats_self_managed FROM servers")

	if err != nil {
		return err
	}

	var targets []syncTarget

	for rows.Next() {
		var t syncTarget

		if err := rows.Scan(&t.serverID, &t.statsSelfManaged); err != nil {
			rows.Close()
			return err
		}

		targets = append(targets, t)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	for _, target := range targets {
		serverID := target.serverID
		guildID, err := snowflake.Parse(serverID)

		if err != nil {
			continue
		}

		guild, err := dclient.Get().Rest().GetGuild(guildID, true)

		if err != nil {
			continue
		}

		avatar := ""

		if url := guild.IconURL(); url != nil {
			avatar = *url
		}

		if target.statsSelfManaged {
			if _, err := state.Pool.Exec(ctx,
				"UPDATE servers SET avatar = $2 WHERE server_id = $1",
				serverID, avatar,
			); err != nil {
				return err
			}
			continue
		}

		if _, err := state.Pool.Exec(ctx,
			"UPDATE servers SET avatar = $2, total_members = $3, online_members = $4 WHERE server_id = $1",
			serverID, avatar, guild.ApproximateMemberCount, guild.ApproximatePresenceCount,
		); err != nil {
			return err
		}
	}

	return nil
}
