package types

import (
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

const (
	MaxBotChangelogTitleLen   = 200
	MaxBotChangelogContentLen = 10000
	MaxBotChangelogVersionLen = 50
)

// @ci table=bot_changelogs, unfilled=1
type BotChangelog struct {
	ID        string    `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	Content   string    `db:"content" json:"content"`
	Version   string    `db:"version" json:"version"`
	CreatedBy string    `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type BotChangelogList struct {
	Changelogs []BotChangelog `json:"changelogs"`
}

type CreateBotChangelog struct {
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1,max=10000"`
	Version string `json:"version" validate:"max=50"`
}

// A single changelog entry in the sitewide feed (GET /bots/@changelogs),
// with just enough of the owning bot's identity to display and link back
// to it.
type BotChangelogFeedEntry struct {
	ID        string                  `db:"id" json:"id"`
	BotID     string                  `db:"bot_id" json:"bot_id"`
	Title     string                  `db:"title" json:"title"`
	Content   string                  `db:"content" json:"content"`
	Version   string                  `db:"version" json:"version"`
	CreatedAt time.Time               `db:"created_at" json:"created_at"`
	User      *dovetypes.PlatformUser `db:"-" json:"user" description:"The bot's user information" ci:"internal"` // Must be parsed internally
}
