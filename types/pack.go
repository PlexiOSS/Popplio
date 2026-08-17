package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// PackType enumerates what a pack bundles. Immutable once a pack is
// created — bot/server/emoji packs are structurally different enough
// (different content arrays entirely) that allowing a pack to change type
// after creation would invite half-migrated rows; an owner who wants a
// different type deletes and recreates instead.
const (
	PackTypeBot    = "bot"
	PackTypeServer = "server"
	PackTypeEmoji  = "emoji"
)

// @ci table=packs, unfilled=1
//
// Represents a Bot Pack
type BotPack struct {
	Owner           string                  `db:"owner" json:"-" description:"The owner of the pack"`
	ResolvedOwner   *dovetypes.PlatformUser `db:"-" json:"owner" ci:"internal" description:"The resolved owner of the pack"` // Owner must be resolved internally from the owner field
	Name            string                  `db:"name" json:"name" description:"The pack's name"`
	Short           string                  `db:"short" json:"short" description:"The pack's short description"`
	Votes           int                     `db:"-" json:"votes" description:"The pack's vote count" ci:"internal"` // Votes are retrieved from entity_votes
	Tags            []string                `db:"tags" json:"tags" description:"The pack's tags"`
	URL             string                  `db:"url" json:"url" description:"The pack's URL"`
	CreatedAt       time.Time               `db:"created_at" json:"created_at" description:"The pack's creation date"`
	PackType        string                  `db:"pack_type" json:"pack_type" description:"What the pack bundles: bot, server, or emoji"`
	Bots            []string                `db:"bots" json:"bot_ids" description:"The pack's bot IDs (pack_type=bot only)"`
	ResolvedBots    []IndexBot              `db:"-" json:"bots" ci:"internal" description:"The resolved bots in the pack"` // Bots must be resolved internally from their IDs
	Servers         []string                `db:"servers" json:"server_ids" description:"The pack's server IDs (pack_type=server only)"`
	ResolvedServers []IndexServer           `db:"-" json:"servers" ci:"internal" description:"The resolved servers in the pack"` // Servers must be resolved internally from their IDs
	Emojis          []PackEmoji             `db:"-" json:"emojis" ci:"internal" description:"The pack's emojis (pack_type=emoji only), resolved from pack_emojis"` // Emojis must be resolved internally from pack_emojis
	VoteBanned      bool                    `db:"vote_banned" json:"vote_banned" description:"Whether the pack is banned from voting"`
}

// PackEmoji is a single emoji bundled into an emoji pack. Unlike
// Server.Emojis (a live, periodically-resynced copy of a real guild's
// emojis), a pack emoji is a durable asset uploaded specifically for the
// pack — it has to keep working even if the source server that inspired it
// stops syncing or leaves the platform entirely, so it can't be a live
// reference into servers.emojis.
//
// Like bot/server banners, there is no asset URL field here — the image
// lives at a deterministic CDN path the frontend builds itself from
// (pack URL, emoji ID, animated), the same convention bannerUrl() already
// uses. Popplio only needs to know the emoji exists.
type PackEmoji struct {
	ID       string `db:"id" json:"id" description:"The emoji's ID within the pack"`
	Name     string `db:"name" json:"name" description:"The emoji's name/shortcode as shown in the pack"`
	Animated bool   `db:"animated" json:"animated" description:"Whether the emoji is an animated GIF"`
	Position int    `db:"position" json:"position" description:"Display order within the pack"`
}

// PackEmojiInput is a single emoji submitted at pack create/edit time. The
// emoji image itself must already exist in the bucket by this point — the
// client uploads each emoji first (keyed by ID, under the pack's URL) and
// only then submits the pack with the IDs it used, so pack writes stay a
// single atomic insert instead of a multi-step transaction reaching out to
// storage. Shared between add_pack and patch_pack rather than duplicated,
// since endpoint packages don't import each other in this codebase.
type PackEmojiInput struct {
	ID       string `json:"id" validate:"required,uuid" msg:"Each emoji needs a valid ID"`
	Name     string `json:"name" validate:"required,min=1,max=32,notblank" msg:"Emoji names must be between 1 and 32 characters"`
	Animated bool   `json:"animated"`
}
