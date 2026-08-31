package types

import (
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

// @ci table=server_templates
type ServerTemplate struct {
	ID         string                  `db:"id" json:"id"`
	Code       string                  `db:"code" json:"code" description:"The Discord server template code"`
	Name       string                  `db:"name" json:"name" description:"The template's name, pulled from Discord at submission time"`
	Short      string                  `db:"short" json:"short" description:"The submitter's description of the template"`
	Tags       []string                `db:"tags" json:"tags" description:"The template's tags"`
	NSFW       bool                    `db:"nsfw" json:"nsfw" description:"Whether the template is for an NSFW-oriented server"`
	OwnerID    string                  `db:"owner" json:"-"`
	Owner      *dovetypes.PlatformUser `db:"-" json:"owner" description:"The user who submitted the template" ci:"internal"` // OwnerID must be resolved internally
	UsageCount int                     `db:"usage_count" json:"usage_count" description:"Discord's own reported usage count, as of submission time"`
	CreatedAt  time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time               `db:"updated_at" json:"updated_at"`
}
