// Package get_list_stats implements GET /list/stats — "Get List Statistics".
//
// Gets basic statistics of the list
package get_list_stats

import (
	"net/http"

	"popplio/api/resp"

	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Statistics",
		Description: "Gets basic statistics of the list",
		Resp:        types.ListStats{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var totalBots int64
	err := state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&totalBots)

	if err != nil {
		return resp.Err("Failed to fetch bot count", err)
	}

	var totalApprovedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'approved'").Scan(&totalApprovedBots)

	if err != nil {
		return resp.Err("Failed to fetch approved bot count", err)
	}

	var totalCertifiedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'certified'").Scan(&totalCertifiedBots)

	if err != nil {
		return resp.Err("Failed to fetch certified bot count", err)
	}

	var totalPendingBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'pending'").Scan(&totalPendingBots)

	if err != nil {
		return resp.Err("Failed to fetch pending bot count", err)
	}

	var totalDeniedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE type = 'denied'").Scan(&totalDeniedBots)

	if err != nil {
		return resp.Err("Failed to fetch denied bot count", err)
	}

	var totalStaff int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM staff_members").Scan(&totalStaff)

	if err != nil {
		return resp.Err("Failed to fetch user count", err)
	}

	var totalUsers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

	if err != nil {
		return resp.Err("Failed to fetch total user count", err)
	}

	var totalVotes int64
	err = state.Pool.QueryRow(d.Context, "SELECT SUM(approximate_votes) FROM bots").Scan(&totalVotes)

	if err != nil {
		return resp.Err("Failed to fetch total vote count", err)
	}

	var totalPacks int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&totalPacks)

	if err != nil {
		return resp.Err("Failed to fetch total pack count", err)
	}

	var totalTickets int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM tickets").Scan(&totalTickets)

	if err != nil {
		return resp.Err("Failed to fetch total ticket count", err)
	}

	var totalBannedUsers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM users WHERE banned = true").Scan(&totalBannedUsers)

	if err != nil {
		return resp.Err("Failed to fetch banned user count", err)
	}

	var totalVoteBannedBots int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots WHERE vote_banned = true").Scan(&totalVoteBannedBots)

	if err != nil {
		return resp.Err("Failed to fetch vote-banned bot count", err)
	}

	var totalServers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers").Scan(&totalServers)

	if err != nil {
		return resp.Err("Failed to fetch server count", err)
	}

	var totalApprovedServers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE type = 'approved'").Scan(&totalApprovedServers)

	if err != nil {
		return resp.Err("Failed to fetch approved server count", err)
	}

	var totalCertifiedServers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE type = 'certified'").Scan(&totalCertifiedServers)

	if err != nil {
		return resp.Err("Failed to fetch certified server count", err)
	}

	var totalPendingServers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE type = 'pending'").Scan(&totalPendingServers)

	if err != nil {
		return resp.Err("Failed to fetch pending server count", err)
	}

	var totalDeniedServers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE type = 'denied'").Scan(&totalDeniedServers)

	if err != nil {
		return resp.Err("Failed to fetch denied server count", err)
	}

	var totalVoteBannedServers int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE vote_banned = true").Scan(&totalVoteBannedServers)

	if err != nil {
		return resp.Err("Failed to fetch vote-banned server count", err)
	}

	return uapi.HttpResponse{
		Json: types.ListStats{
			TotalBots:              totalBots,
			TotalApprovedBots:      totalApprovedBots,
			TotalCertifiedBots:     totalCertifiedBots,
			TotalPendingBots:       totalPendingBots,
			TotalDeniedBots:        totalDeniedBots,
			TotalStaff:             totalStaff,
			TotalUsers:             totalUsers,
			TotalVotes:             totalVotes,
			TotalPacks:             totalPacks,
			TotalTickets:           totalTickets,
			TotalBannedUsers:       totalBannedUsers,
			TotalVoteBannedBots:    totalVoteBannedBots,
			TotalServers:           totalServers,
			TotalApprovedServers:   totalApprovedServers,
			TotalCertifiedServers:  totalCertifiedServers,
			TotalPendingServers:    totalPendingServers,
			TotalDeniedServers:     totalDeniedServers,
			TotalVoteBannedServers: totalVoteBannedServers,
		},
	}
}
