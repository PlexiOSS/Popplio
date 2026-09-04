package types

import (
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

const (
	PackTypeBot     = "bot"
	PackTypeServer  = "server"
	PackTypeEmoji   = "emoji"
	PackTypeSticker = "sticker"
	PackTypeSound   = "sound"
)

// @ci table=packs, unfilled=1
type BotPack struct {
	Owner           string                  `db:"owner" json:"-" description:"The owner of the pack"`
	ResolvedOwner   *dovetypes.PlatformUser `db:"-" json:"owner" ci:"internal" description:"The resolved owner of the pack"` // Owner must be resolved internally from the owner field
	Name            string                  `db:"name" json:"name" description:"The pack's name"`
	Short           string                  `db:"short" json:"short" description:"The pack's short description"`
	Votes           int                     `db:"-" json:"votes" description:"The pack's vote count" ci:"internal"` // Votes are retrieved from entity_votes
	Tags            []string                `db:"tags" json:"tags" description:"The pack's tags"`
	URL             string                  `db:"url" json:"url" description:"The pack's URL"`
	CreatedAt       time.Time               `db:"created_at" json:"created_at" description:"The pack's creation date"`
	PackType        string                  `db:"pack_type" json:"pack_type" description:"What the pack bundles: bot, server, emoji, or sticker"`
	Bots            []string                `db:"bots" json:"bot_ids" description:"The pack's bot IDs (pack_type=bot only)"`
	ResolvedBots    []IndexBot              `db:"-" json:"bots" ci:"internal" description:"The resolved bots in the pack"` // Bots must be resolved internally from their IDs
	Servers         []string                `db:"servers" json:"server_ids" description:"The pack's server IDs (pack_type=server only)"`
	ResolvedServers []IndexServer           `db:"-" json:"servers" ci:"internal" description:"The resolved servers in the pack"`                                   // Servers must be resolved internally from their IDs
	Emojis          []PackEmoji             `db:"-" json:"emojis" ci:"internal" description:"The pack's emojis (pack_type=emoji only), resolved from pack_emojis"` // Emojis must be resolved internally from pack_emojis
	Stickers        []PackSticker           `db:"-" json:"stickers" ci:"internal" description:"The pack's stickers (pack_type=sticker only), resolved from pack_stickers"`
	Sounds          []PackSound             `db:"-" json:"sounds" ci:"internal" description:"The pack's sounds (pack_type=sound only), resolved from pack_sounds"`
	VoteBanned      bool                    `db:"vote_banned" json:"vote_banned" description:"Whether the pack is banned from voting"`
}

type PackEmoji struct {
	ID        string `db:"id" json:"id" description:"The emoji's ID within the pack"`
	Name      string `db:"name" json:"name" description:"The emoji's name/shortcode as shown in the pack"`
	Animated  bool   `db:"animated" json:"animated" description:"Whether the emoji is an animated GIF"`
	Position  int    `db:"position" json:"position" description:"Display order within the pack"`
	Downloads int    `db:"downloads" json:"downloads" description:"How many times this specific emoji has been individually downloaded from its own page"`
	Vanity    string `db:"-" json:"vanity" description:"The emoji's own short vanity code" ci:"internal"`
}

type PackEmojiInput struct {
	ID       string `json:"id" validate:"required,uuid" msg:"Each emoji needs a valid ID"`
	Name     string `json:"name" validate:"required,min=1,max=32,notblank" msg:"Emoji names must be between 1 and 32 characters"`
	Animated bool   `json:"animated"`
}

type PackSticker struct {
	ID        string `db:"id" json:"id" description:"The sticker's ID within the pack"`
	Name      string `db:"name" json:"name" description:"The sticker's name"`
	Animated  bool   `db:"animated" json:"animated" description:"Whether the sticker is an animated GIF"`
	Position  int    `db:"position" json:"position" description:"Display order within the pack"`
	Downloads int    `db:"downloads" json:"downloads" description:"How many times this specific sticker has been individually downloaded from its own page"`
	Vanity    string `db:"-" json:"vanity" description:"The sticker's own short vanity code" ci:"internal"`
}

// PackStickerInput is PackEmojiInput's counterpart for sticker packs.
type PackStickerInput struct {
	ID       string `json:"id" validate:"required,uuid" msg:"Each sticker needs a valid ID"`
	Name     string `json:"name" validate:"required,min=1,max=32,notblank" msg:"Sticker names must be between 1 and 32 characters"`
	Animated bool   `json:"animated"`
}

type PackSound struct {
	ID         string `db:"id" json:"id" description:"The sound's ID within the pack"`
	Name       string `db:"name" json:"name" description:"The sound's name"`
	DurationMs int    `db:"duration_ms" json:"duration_ms" description:"The sound clip's duration, in milliseconds"`
	Position   int    `db:"position" json:"position" description:"Display order within the pack"`
	Downloads  int    `db:"downloads" json:"downloads" description:"How many times this specific sound has been individually downloaded from its own page"`
	Vanity     string `db:"-" json:"vanity" description:"The sound's own short vanity code" ci:"internal"`
}

// PackSoundInput is PackEmojiInput's counterpart for sound packs.
type PackSoundInput struct {
	ID         string `json:"id" validate:"required,uuid" msg:"Each sound needs a valid ID"`
	Name       string `json:"name" validate:"required,min=1,max=32,notblank" msg:"Sound names must be between 1 and 32 characters"`
	DurationMs int    `json:"duration_ms" validate:"min=0" msg:"Duration cannot be negative"`
}

// FlatPackEmoji is one row of the flat, sitewide GET /emojis/@all browse
// feed -- PackEmoji plus just enough of its owning pack to link back to it.
type FlatPackEmoji struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Animated  bool      `json:"animated"`
	Downloads int       `json:"downloads"`
	CreatedAt time.Time `json:"created_at"`
	PackURL   string    `json:"pack_url"`
	PackName  string    `json:"pack_name"`
}

// FlatPackSticker is FlatPackEmoji's counterpart for GET /stickers/@all.
type FlatPackSticker struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Animated  bool      `json:"animated"`
	Downloads int       `json:"downloads"`
	CreatedAt time.Time `json:"created_at"`
	PackURL   string    `json:"pack_url"`
	PackName  string    `json:"pack_name"`
}

// FlatPackSound is FlatPackEmoji's counterpart for GET /sounds/@all.
type FlatPackSound struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	DurationMs int       `json:"duration_ms"`
	Downloads  int       `json:"downloads"`
	CreatedAt  time.Time `json:"created_at"`
	PackURL    string    `json:"pack_url"`
	PackName   string    `json:"pack_name"`
}

// PackEmojiDetail is GET /emojis/{id}'s response -- a single emoji plus its
// owning pack's identity and the pack owner, resolved.
type PackEmojiDetail struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Animated  bool                    `json:"animated"`
	Position  int                     `json:"position"`
	Downloads int                     `json:"downloads"`
	CreatedAt time.Time               `json:"created_at"`
	PackURL   string                  `json:"pack_url"`
	PackName  string                  `json:"pack_name"`
	Owner     *dovetypes.PlatformUser `json:"owner"`
	// Vanity is the emoji's own short vanity code, if the owner has set one
	// (via PATCH /pack_emoji/{id}/vanity). Empty when unset -- unlike
	// bots/servers/teams, pack emojis aren't required to have one.
	Vanity string `json:"vanity"`
}

// PackStickerDetail is PackEmojiDetail's counterpart for GET /stickers/{id}.
type PackStickerDetail struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Animated  bool                    `json:"animated"`
	Position  int                     `json:"position"`
	Downloads int                     `json:"downloads"`
	CreatedAt time.Time               `json:"created_at"`
	PackURL   string                  `json:"pack_url"`
	PackName  string                  `json:"pack_name"`
	Owner     *dovetypes.PlatformUser `json:"owner"`
	Vanity    string                  `json:"vanity"`
}

// PackSoundDetail is PackEmojiDetail's counterpart for GET /sounds/{id}.
type PackSoundDetail struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	DurationMs int                     `json:"duration_ms"`
	Position   int                     `json:"position"`
	Downloads  int                     `json:"downloads"`
	CreatedAt  time.Time               `json:"created_at"`
	PackURL    string                  `json:"pack_url"`
	PackName   string                  `json:"pack_name"`
	Owner      *dovetypes.PlatformUser `json:"owner"`
	Vanity     string                  `json:"vanity"`
}
