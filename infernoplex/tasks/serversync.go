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
			// disgo's Emoji.URL() always defaults to PNG regardless of
			// e.Animated — unlike Sticker.URL(), which already infers the
			// right format from its FormatType. Has to be requested
			// explicitly here or animated emojis get a permanently-static
			// URL despite being flagged as animated.
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

func syncServerMeta(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, "SELECT server_id FROM servers")

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

		// The gateway-cached guild (dclient.Get().Caches().Guild) only tracks
		// member/presence counts if the privileged Server Members intent is
		// enabled, which this bot deliberately doesn't request (see
		// TeamCleanup's doc comment) — so its MemberCount is just whatever
		// was in the initial GUILD_CREATE payload and never updates. A REST
		// fetch with with_counts=true gets a live approximate count without
		// needing that intent, the same way /setup does on server creation.
		guild, err := dclient.Get().Rest().GetGuild(guildID, true)

		if err != nil {
			continue
		}

		avatar := ""

		if url := guild.IconURL(); url != nil {
			avatar = *url
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
