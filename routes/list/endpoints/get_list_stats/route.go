// Package get_list_stats implements GET /list/stats — "Get List Statistics".
//
// Gets basic statistics of the list
package get_list_stats

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
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
	row, err := db.New(state.Pool).GetListStats(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch list stats", err)
	}

	return uapi.HttpResponse{
		Json: types.ListStats{
			TotalBots:              row.TotalBots,
			TotalApprovedBots:      row.TotalApprovedBots,
			TotalCertifiedBots:     row.TotalCertifiedBots,
			TotalPendingBots:       row.TotalPendingBots,
			TotalDeniedBots:        row.TotalDeniedBots,
			TotalStaff:             row.TotalStaff,
			TotalUsers:             row.TotalUsers,
			TotalVotes:             row.TotalVotes,
			TotalPacks:             row.TotalPacks,
			TotalTickets:           row.TotalTickets,
			TotalBannedUsers:       row.TotalBannedUsers,
			TotalVoteBannedBots:    row.TotalVoteBannedBots,
			TotalServers:           row.TotalServers,
			TotalApprovedServers:   row.TotalApprovedServers,
			TotalCertifiedServers:  row.TotalCertifiedServers,
			TotalPendingServers:    row.TotalPendingServers,
			TotalDeniedServers:     row.TotalDeniedServers,
			TotalVoteBannedServers: row.TotalVoteBannedServers,
		},
	}
}
