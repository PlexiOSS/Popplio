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
	Channels   []TemplateChannel       `db:"channels" json:"channels" description:"A preview of the template's channels, captured from Discord at submission time. Empty on list responses -- only the single-template fetch populates this"`
	Roles      []TemplateRole          `db:"roles" json:"roles" description:"A preview of the template's roles (excluding @everyone), captured from Discord at submission time. Empty on list responses -- only the single-template fetch populates this"`
	Likes      int                     `db:"-" json:"likes" ci:"internal"`
	Dislikes   int                     `db:"-" json:"dislikes" ci:"internal"`
}

type TemplateChannel struct {
	ID       int    `json:"id" description:"The channel's ID within the template (not a real Discord snowflake -- scoped to this template only)"`
	Type     int    `json:"type" description:"Discord's channel type enum (0=text, 2=voice, 4=category, 5=announcement, 13=stage, 15=forum, 16=media)"`
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id" description:"The category channel's ID this channel sits under, if any"`
	Position int    `json:"position"`
}

// TemplateRole is a stripped-down preview of one of a template's roles --
// name and color only, no permission breakdown.
type TemplateRole struct {
	ID    int    `json:"id" description:"The role's ID within the template (not a real Discord snowflake -- scoped to this template only)"`
	Name  string `json:"name"`
	Color int    `json:"color" description:"The role's color as a decimal RGB integer, 0 if none"`
}

// TemplateReactionSummary is the response shape for both reading and
// setting a template's like/dislike reaction -- see
// server_template_reactions' own migration comment for why this isn't
// modeled as a vote.
type TemplateReactionSummary struct {
	Likes     int   `json:"likes"`
	Dislikes  int   `json:"dislikes"`
	UserLiked *bool `json:"user_liked" description:"The requesting user's own reaction: true if liked, false if disliked, null if they haven't reacted"`
}
