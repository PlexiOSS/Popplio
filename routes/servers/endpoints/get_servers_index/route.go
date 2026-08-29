package get_servers_index

import (
	"context"
	"fmt"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Servers Index",
		Description: "Gets the index of the server-side of the list. Returns a ``ListIndexServer`` object",
		Resp:        types.ListIndexServer{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listIndex := types.ListIndexServer{}

	q := db.New(state.Pool)

	certRows, err := q.GetCertifiedIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting certified servers", err)
	}
	listIndex.Certified, err = processRow(d.Context, toIndexServersFromCertified(certRows))
	if err != nil {
		return resp.Err("Error while processing certified servers", err)
	}

	premRows, err := q.GetPremiumIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting premium servers", err)
	}
	listIndex.Premium, err = processRow(d.Context, toIndexServersFromPremium(premRows))
	if err != nil {
		return resp.Err("Error while processing premium servers", err)
	}

	mostViewedRows, err := q.GetMostViewedIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting most viewed servers", err)
	}
	listIndex.MostViewed, err = processRow(d.Context, toIndexServersFromMostViewed(mostViewedRows))
	if err != nil {
		return resp.Err("Error while processing most viewed servers", err)
	}

	recentlyAddedRows, err := q.GetRecentlyAddedIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting recently added servers", err)
	}
	listIndex.RecentlyAdded, err = processRow(d.Context, toIndexServersFromRecentlyAdded(recentlyAddedRows))
	if err != nil {
		return resp.Err("Error while processing recently added servers", err)
	}

	topVotedRows, err := q.GetTopVotedIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting top voted servers", err)
	}
	listIndex.TopVoted, err = processRow(d.Context, toIndexServersFromTopVoted(topVotedRows))
	if err != nil {
		return resp.Err("Error while processing top voted servers", err)
	}

	featuredRows, err := q.GetFeaturedIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting featured servers", err)
	}
	listIndex.Featured, err = processRow(d.Context, toIndexServersFromFeatured(featuredRows))
	if err != nil {
		return resp.Err("Error while processing featured servers", err)
	}

	// Spotlight Servers (set by staff, distinct from shop-purchased Featured)
	spotlightRows, err := q.GetSpotlightIndexServers(d.Context)
	if err != nil {
		return resp.Err("Error while getting spotlight servers", err)
	}
	listIndex.Spotlight, err = processRow(d.Context, toIndexServersFromSpotlight(spotlightRows))
	if err != nil {
		return resp.Err("Error while processing spotlight servers", err)
	}

	return uapi.HttpResponse{
		Json: listIndex,
	}
}

// processRow validates that every returned server actually matches the
// public/approved-or-certified invariant every one of these queries relies
// on, then resolves each server (vanity, votes) concurrently.
func processRow(ctx context.Context, servers []types.IndexServer) ([]types.IndexServer, error) {
	for i := range servers {
		if (servers[i].Type != "approved" && servers[i].Type != "certified") || servers[i].State != "public" {
			return nil, fmt.Errorf("internal error: servers %s has invalid type %s or state %s", servers[i].ServerID, servers[i].Type, servers[i].State)
		}
	}

	if err := assets.ResolveIndexServers(ctx, servers); err != nil {
		return nil, err
	}

	return servers, nil
}

func toIndexServersFromCertified(rows []db.GetCertifiedIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}

func toIndexServersFromPremium(rows []db.GetPremiumIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}

func toIndexServersFromMostViewed(rows []db.GetMostViewedIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}

func toIndexServersFromRecentlyAdded(rows []db.GetRecentlyAddedIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}

func toIndexServersFromTopVoted(rows []db.GetTopVotedIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}

func toIndexServersFromFeatured(rows []db.GetFeaturedIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}

func toIndexServersFromSpotlight(rows []db.GetSpotlightIndexServersRow) []types.IndexServer {
	servers := make([]types.IndexServer, len(rows))
	for i, row := range rows {
		servers[i] = types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}
	}
	return servers
}
