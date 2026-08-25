package types

// List Stats

type ListStats struct {
	TotalBots           int64 `json:"total_bots" description:"The list of all bots on the list as ListStatsBot objects (partial bot objects)"`
	TotalApprovedBots   int64 `json:"total_approved_bots" description:"The total number of approved bots on the list"`
	TotalCertifiedBots  int64 `json:"total_certified_bots" description:"The total number of certified bots on the list"`
	TotalPendingBots    int64 `json:"total_pending_bots" description:"The total number of bots awaiting review"`
	TotalDeniedBots     int64 `json:"total_denied_bots" description:"The total number of bots denied on review"`
	TotalStaff          int64 `json:"total_staff" description:"The total number of staff members on the list"`
	TotalUsers          int64 `json:"total_users" description:"The total number of users on the list"`
	TotalVotes          int64 `json:"total_votes" description:"The total number of votes on the list"`
	TotalPacks          int64 `json:"total_packs" description:"The total number of packs on the list"`
	TotalTickets        int64 `json:"total_tickets" description:"The total number of tickets created on the list"`
	TotalBannedUsers    int64 `json:"total_banned_users" description:"The total number of users banned from the list"`
	TotalVoteBannedBots int64 `json:"total_vote_banned_bots" description:"The total number of bots banned from voting"`

	TotalServers           int64 `json:"total_servers" description:"The total number of servers on the list"`
	TotalApprovedServers   int64 `json:"total_approved_servers" description:"The total number of approved servers on the list"`
	TotalCertifiedServers  int64 `json:"total_certified_servers" description:"The total number of certified servers on the list"`
	TotalPendingServers    int64 `json:"total_pending_servers" description:"The total number of servers awaiting review"`
	TotalDeniedServers     int64 `json:"total_denied_servers" description:"The total number of servers denied on review"`
	TotalVoteBannedServers int64 `json:"total_vote_banned_servers" description:"The total number of servers banned from voting"`
}
