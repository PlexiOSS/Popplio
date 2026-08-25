package types

import (
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

// @ci table=changelogs, unfilled=1
type ChangelogEntry struct {
	Itag             string                  `db:"itag" json:"itag" description:"The unique id of the changelog entry"`
	Project          string                  `db:"project" json:"project" description:"Which project this entry belongs to: 'popplio', 'omniplex', or 'keel'"`
	Version          string                  `db:"version" json:"version" description:"The version this entry is for"`
	Added            []string                `db:"added" json:"added" description:"What was added in this version"`
	Updated          []string                `db:"updated" json:"updated" description:"What was changed in this version"`
	Fixed            []string                `db:"fixed" json:"fixed" description:"What was fixed in this version"`
	Removed          []string                `db:"removed" json:"removed" description:"What was removed in this version"`
	ExtraDescription string                  `db:"extra_description" json:"extra_description" description:"Additional freeform notes about this release"`
	Prerelease       bool                    `db:"prerelease" json:"prerelease" description:"Whether this is a prerelease"`
	CreatedBy        string                  `db:"created_by" json:"-"` // Must be parsed internally
	Author           *dovetypes.PlatformUser `db:"-" json:"author" description:"The staff member who authored this entry"`
	CreatedAt        time.Time               `db:"created_at" json:"created_at" description:"When this entry was published"`
}

type ChangelogList struct {
	Entries []ChangelogEntry `json:"entries" description:"The list of published changelog entries"`
}
