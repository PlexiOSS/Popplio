package resolvers

import (
	"context"
	"fmt"

	"github.com/PlexiOSS/Keel/uuidutil"
	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	serverAssets "popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/PlexiOSS/Keel/dovewing"
)

func GetTeamEntities(ctx context.Context, teamId string, targets []string) (*types.TeamEntities, error) {

	eto := &types.TeamEntities{Targets: []string{}}

	q := db.New(state.Pool)

	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamId); err != nil {
		return nil, fmt.Errorf("invalid team id: %w", err)
	}

	for _, st := range targets {
		var isInvalid bool
		switch st {
		case "team_member":
			memberRows, err := q.GetTeamMembersByTeamID(ctx, teamUUID)

			if err != nil {
				return nil, err
			}

			eto.Members = make([]types.TeamMember, len(memberRows))
			for i, row := range memberRows {
				eto.Members[i] = types.TeamMember{
					ITag:        row.Itag,
					TeamID:      uuidutil.Encode(row.TeamID.Bytes),
					UserID:      row.UserID,
					Flags:       row.Flags,
					Service:     row.Service,
					CreatedAt:   row.CreatedAt.Time,
					Mentionable: row.Mentionable,
					DataHolder:  row.DataHolder,
				}
			}

			for i := range eto.Members {
				eto.Members[i].User, err = dovewing.GetUser(ctx, eto.Members[i].UserID, state.DovewingPlatformDiscord)

				if err != nil {
					return nil, err
				}
			}
		case "bot":
			indexBotRows, err := q.GetIndexBotsByTeamOwner(ctx, teamUUID)

			if err != nil {
				return nil, err
			}

			eto.Bots = make([]types.IndexBot, len(indexBotRows))
			for i, row := range indexBotRows {
				eto.Bots[i] = types.IndexBot{
					BotID:            row.BotID,
					Short:            row.Short,
					Type:             row.Type,
					VanityRef:        row.VanityRef,
					ApproximateVotes: int(row.ApproximateVotes),
					Shards:           int(row.Shards),
					Library:          row.Library,
					InviteClick:      int(row.InviteClicks),
					Clicks:           int(row.Clicks),
					Servers:          int(row.Servers),
					NSFW:             row.Nsfw,
					Tags:             row.Tags,
					Premium:          row.Premium,
					CreatedAt:        row.CreatedAt,
					SelfStatus:       row.SelfStatus,
					LastStatsPost:    row.LastStatsPost,
					SupporterBadge:   row.SupporterBadge,
					BoostedUntil:     row.BoostedUntil,
					FeaturedUntil:    row.FeaturedUntil,
					SpotlightedUntil: row.SpotlightedUntil,
					VoteBlitzUntil:   row.VoteBlitzUntil,
				}
			}

			if err := botAssets.ResolveIndexBots(ctx, eto.Bots); err != nil {
				return nil, fmt.Errorf("error occurred while resolving index bot: %w", err)
			}
		case "server":
			indexServerRows, err := q.GetIndexServersByTeamOwner(ctx, teamUUID)

			if err != nil {
				return nil, err
			}

			eto.Servers = make([]types.IndexServer, len(indexServerRows))
			for i, row := range indexServerRows {
				eto.Servers[i] = types.IndexServer{
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

			if err := serverAssets.ResolveIndexServers(ctx, eto.Servers); err != nil {
				return nil, fmt.Errorf("error occurred while resolving index server: %w", err)
			}
		default:
			isInvalid = true
		}

		if !isInvalid {
			eto.Targets = append(eto.Targets, st)
		}
	}

	return eto, nil
}
