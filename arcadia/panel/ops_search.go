// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
	"popplio/votes"
)

type searchServerRow struct {
	ServerID             string
	Name                 string
	Avatar               string
	TotalMembers         int32
	OnlineMembers        int32
	Short                string
	Type                 string
	ApprovalNote         string
	ApproximateVotes     int32
	InviteClicks         int32
	Clicks               int32
	NSFW                 bool
	DiscordNSFWLevel     int32
	NSFWChannelCount     int32
	Tags                 []string
	Premium              bool
	ClaimedBy            *string
	LastClaimed          *time.Time
	ModerationFlagged    bool
	ModerationCategories []string
}

func (s *Server) searchEntitys(ctx context.Context, q *types.QSearchEntitys) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	pattern := "%" + q.Query + "%"
	queries := db.New(state.Pool)

	switch q.TargetType {
	case types.TargetTypeBot:
		rows, err := queries.SearchBotsQueue(ctx, db.SearchBotsQueueParams{Query: q.Query, Pattern: pattern})

		if err != nil {
			return response{}, newError(err)
		}

		queue := make([]botQueueRow, len(rows))

		for i, row := range rows {
			var claimedBy *string

			if row.ClaimedBy.Valid {
				claimedBy = &row.ClaimedBy.String
			}

			queue[i] = botQueueRow{
				BotID:                row.BotID,
				ClientID:             row.ClientID,
				LastClaimed:          timePtr(row.LastClaimed),
				ClaimedBy:            claimedBy,
				Type:                 row.Type,
				ApprovalNote:         row.ApprovalNote,
				Short:                row.Short,
				Invite:               row.Invite,
				ApproximateVotes:     row.ApproximateVotes,
				Shards:               row.Shards,
				Library:              row.Library,
				InviteClicks:         row.InviteClicks,
				Clicks:               row.Clicks,
				Servers:              row.Servers,
				ModerationFlagged:    row.ModerationFlagged,
				ModerationCategories: row.ModerationCategories,
			}
		}

		return s.partialBots(ctx, queue)
	case types.TargetTypeServer:
		rows, err := queries.SearchServersQueue(ctx, db.SearchServersQueueParams{Query: q.Query, Pattern: pattern})

		if err != nil {
			return response{}, newError(err)
		}

		queue := make([]searchServerRow, len(rows))

		for i, row := range rows {
			var claimedBy *string

			if row.ClaimedBy.Valid {
				claimedBy = &row.ClaimedBy.String
			}

			queue[i] = searchServerRow{
				ServerID:             row.ServerID,
				Name:                 row.Name,
				Avatar:               row.Avatar,
				TotalMembers:         row.TotalMembers,
				OnlineMembers:        row.OnlineMembers,
				Short:                row.Short,
				Type:                 row.Type,
				ApprovalNote:         row.ApprovalNote,
				ApproximateVotes:     row.ApproximateVotes,
				InviteClicks:         row.InviteClicks,
				Clicks:               row.Clicks,
				NSFW:                 row.Nsfw,
				DiscordNSFWLevel:     int32(row.DiscordNsfwLevel),
				NSFWChannelCount:     row.NsfwChannelCount,
				Tags:                 row.Tags,
				Premium:              row.Premium,
				ClaimedBy:            claimedBy,
				LastClaimed:          timePtr(row.LastClaimed),
				ModerationFlagged:    row.ModerationFlagged,
				ModerationCategories: row.ModerationCategories,
			}
		}

		return s.partialServers(ctx, queue)
	case types.TargetTypePack:
		queue, err := queries.SearchPacksQueue(ctx, db.SearchPacksQueueParams{Query: q.Query, Pattern: pattern})

		if err != nil {
			return response{}, newError(err)
		}

		packs := make([]types.PartialEntity, 0, len(queue))

		for _, pack := range queue {
			owner, err := impls.GetPlatformUser(ctx, pack.Owner)

			if err != nil {
				return response{}, newError(err)
			}

			voteCount, err := votes.EntityGetVoteCount(ctx, state.Pool, pack.Url, "pack")

			if err != nil {
				return response{}, newError(err)
			}

			packs = append(packs, types.PartialEntity{Pack: &types.PartialPack{
				URL:        pack.Url,
				Name:       pack.Name,
				Short:      pack.Short,
				PackType:   pack.PackType,
				Owner:      owner,
				Votes:      int32(voteCount),
				Tags:       types.NonNilStrings(pack.Tags),
				VoteBanned: pack.VoteBanned,
			}})
		}

		return writeJSON(http.StatusOK, packs), nil
	case types.TargetTypeTeam:
		queue, err := queries.SearchTeamsQueue(ctx, db.SearchTeamsQueueParams{Query: q.Query, Pattern: pattern})

		if err != nil {
			return response{}, newError(err)
		}

		teams := make([]types.PartialEntity, 0, len(queue))

		for _, team := range queue {
			voteCount, err := votes.EntityGetVoteCount(ctx, state.Pool, team.ID, "team")

			if err != nil {
				return response{}, newError(err)
			}

			teams = append(teams, types.PartialEntity{Team: &types.PartialTeam{
				ID:         team.ID,
				Name:       team.Name,
				Short:      team.Short,
				Votes:      int32(voteCount),
				Tags:       types.NonNilStrings(team.Tags),
				NSFW:       team.Nsfw,
				VoteBanned: team.VoteBanned,
			}})
		}

		return writeJSON(http.StatusOK, teams), nil
	case types.TargetTypeUser:
		queue, err := queries.SearchUsersQueue(ctx, db.SearchUsersQueueParams{Query: q.Query, Pattern: pattern})

		if err != nil {
			return response{}, newError(err)
		}

		results := make([]types.PartialEntity, 0, len(queue))

		for _, u := range queue {
			platformUser, err := impls.GetPlatformUser(ctx, u.UserID)

			if err != nil {
				return response{}, newError(err)
			}

			staffCount, err := queries.CountStaffMemberByID(ctx, u.UserID)

			if err != nil {
				return response{}, newError(err)
			}

			results = append(results, types.PartialEntity{User: &types.PartialUser{
				User:       platformUser,
				Staff:      staffCount > 0,
				Banned:     u.Banned,
				VoteBanned: u.VoteBanned,
			}})
		}

		return writeJSON(http.StatusOK, results), nil
	default:
		return writeText(http.StatusNotImplemented, "Searching this target type is not implemented"), nil
	}
}
