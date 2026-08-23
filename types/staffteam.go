package types

// PublicStaffPosition is the public-safe projection of a staff position —
// name and rank only, never the permission grants attached to it.
type PublicStaffPosition struct {
	Name  string `json:"name" description:"The position's name"`
	Icon  string `json:"icon" description:"The position's icon URL, if any"`
	Index int32  `json:"index" description:"Where this position ranks — lower is more senior"`
}

// PublicStaffMember is the public-safe projection of a staff member for the
// team page. Unlike arcadia/types.StaffMember (the staff-panel shape), this
// deliberately excludes permissions, disciplinaries, and sync/security
// metadata — none of which is anyone's business but staff's.
type PublicStaffMember struct {
	UserID      string                `json:"user_id" description:"The staff member's user ID"`
	Username    string                `json:"username" description:"The staff member's username"`
	DisplayName string                `json:"display_name" description:"The staff member's display name"`
	Avatar      string                `json:"avatar" description:"The staff member's avatar URL"`
	Positions   []PublicStaffPosition `json:"positions" description:"Every position this member holds, most senior first"`
}
