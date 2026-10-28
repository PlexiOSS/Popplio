package types

import "time"

// BadgeCatalog is one row of the badges catalog, as the public API exposes
// it — the same data Arcadia's panel manages, read-only here.
type BadgeCatalog struct {
	ID          string   `json:"id" description:"The badge's ID"`
	Name        string   `json:"name" description:"The badge's display name"`
	Description string   `json:"description" description:"What the badge represents"`
	Icon        string   `json:"icon" description:"A lucide-react icon name"`
	Color       string   `json:"color" description:"One of the site's badge color variants"`
	TargetTypes []string `json:"target_types" description:"Which entity types this badge can be assigned to"`
}

// EntityBadge is one badge assigned to one entity.
type EntityBadge struct {
	Badge     BadgeCatalog `json:"badge"`
	Reason    string       `json:"reason" description:"Why the badge was awarded, if a reason was given"`
	AwardedBy string       `json:"awarded_by" description:"The staff member who awarded the badge"`
	CreatedAt time.Time    `json:"created_at" description:"When the badge was awarded"`
}

type EntityBadgeList struct {
	Badges []EntityBadge `json:"badges"`
}

type PublicBadge struct {
	BadgeCatalog
	Automatic  bool   `json:"automatic" description:"Whether the badge is awarded and removed automatically by a rule"`
	HowToEarn  string `json:"how_to_earn" description:"A plain-English description of how to earn the badge"`
	LinkedRole bool   `json:"linked_role" description:"Whether the badge is also available as a Discord linked role"`
	Holders    int    `json:"holders" description:"How many users, bots, servers and teams currently hold the badge"`
}

type PublicBadgeList struct {
	Badges []PublicBadge `json:"badges"`
}
