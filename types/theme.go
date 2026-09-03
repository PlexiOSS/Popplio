package types

import (
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

// ThemeCategories is the fixed, closed vocabulary a theme's tags are drawn
// from -- unlike bot/server tags, which allow free text, a theme's
// categories are validated against this exact list server-side (see
// add_theme's `oneof` validation).
var ThemeCategories = []string{
	"Green",
	"Blue",
	"Purple",
	"Pink",
	"Red",
	"Orange",
	"Dark",
	"Light",
	"Gradient",
	"Aesthetic",
	"Minimal",
	"Vibrant",
}

// Theme is a Discord profile theme: a name plus two hex colors, submitted
// by a user for others to browse and copy. No file upload, no team
// permissions -- a simply-owned entity, closest in shape to pack
// emojis/stickers.
type Theme struct {
	ID             string                  `db:"id" json:"id"`
	Owner          string                  `db:"owner" json:"-"`
	ResolvedOwner  *dovetypes.PlatformUser `db:"-" json:"owner" ci:"internal"`
	Name           string                  `db:"name" json:"name"`
	PrimaryColor   string                  `db:"primary_color" json:"primary_color"`
	SecondaryColor string                  `db:"secondary_color" json:"secondary_color"`
	Tags           []string                `db:"tags" json:"tags"`
	CreatedAt      time.Time               `db:"created_at" json:"created_at"`
}
