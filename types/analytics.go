// Copyright (C) 2026 NodeByte LTD

package types

type EntityAnalyticsDay struct {
	Day           string `json:"day" description:"The UTC day, as YYYY-MM-DD"`
	Votes         int    `json:"votes" description:"Upvotes cast that day, including automatic ones"`
	AutoVotes     int    `json:"auto_votes" description:"How many of that day's votes were cast by auto-vote"`
	Views         *int   `json:"views" description:"Page views that day, null when there's no snapshot for that day and the day before"`
	InviteClicks  *int   `json:"invite_clicks" description:"Invite clicks that day, null when there's no snapshot for that day and the day before"`
	Servers       *int   `json:"servers,omitempty" description:"Bots only: the bot's server count at the end of the day, null without a snapshot"`
	Users         *int   `json:"users,omitempty" description:"Bots only: the bot's user count at the end of the day, null without a snapshot"`
	TotalMembers  *int   `json:"total_members,omitempty" description:"Servers only: the member count at the end of the day, null without a snapshot"`
	OnlineMembers *int   `json:"online_members,omitempty" description:"Servers only: the online member count at the end of the day, null without a snapshot"`
}

type EntityAnalytics struct {
	TrackedSince *string              `json:"tracked_since" description:"The first day views, invite clicks and counts were recorded, null if not yet. Votes are available for the whole range."`
	Days         []EntityAnalyticsDay `json:"days" description:"One entry per day, oldest first, ending today (UTC)"`
}
