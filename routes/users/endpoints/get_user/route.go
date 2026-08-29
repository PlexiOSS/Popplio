// Package get_user implements GET /users/{id} — "Get User".
//
// Gets a user by id
package get_user

import (
	"encoding/json"
	"errors"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	"popplio/routes/packs/assets"
	serverAssets "popplio/routes/servers/assets"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/votes"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User",
		Description: "Gets a user by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.User{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	userId := chi.URLParam(r, "id")

	if userId == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	q := db.New(state.Pool)

	row, err := q.GetUserByID(d.Context, userId)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Error while getting user", zap.Error(err), zap.String("userID", userId))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	var extraLinks []types.Link
	if err := json.Unmarshal(row.ExtraLinks, &extraLinks); err != nil {
		return resp.Err("Error parsing user extra_links [json]", err, zap.String("userID", userId))
	}

	user := types.User{
		ITag:                  row.Itag,
		ID:                    row.UserID,
		Experiments:           row.Experiments,
		Certified:             row.Certified,
		BotDeveloper:          row.Developer,
		BugHunters:            row.BugHunters,
		CaptchaSponsorEnabled: row.CaptchaSponsorEnabled,
		ExtraLinks:            extraLinks,
		About:                 row.About,
		VoteBanned:            row.VoteBanned,
		Banned:                row.Banned,
		CreatedAt:             row.CreatedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
	}

	if row.LastBoosterClaim.Valid {
		user.LastBoosterClaim = &row.LastBoosterClaim.Time
	}

	userObj, err := dovewing.GetUser(d.Context, user.ID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error while getting user [collect]", err, zap.String("userID", user.ID))
	}

	user.User = userObj

	indexBotRows, err := q.GetIndexBotsByOwner(d.Context, pgtype.Text{String: user.ID, Valid: true})

	if err != nil {
		return resp.Err("Failed to get user bots [db fetch]", err, zap.String("userID", user.ID))
	}

	user.UserBots = make([]types.IndexBot, len(indexBotRows))
	for i, row := range indexBotRows {
		user.UserBots[i] = types.IndexBot{
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

	// Resolve the userbots concurrently, since each bot's resolution is independent
	if err := botAssets.ResolveIndexBots(d.Context, user.UserBots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	// Servers, like bots, but exclusively team-owned there's no direct
	// `owner` column on servers, so this is always via team membership.
	indexServerRows, err := q.GetIndexServersByTeamMembership(d.Context, user.ID)

	if err != nil {
		return resp.Err("Failed to get user servers [db fetch]", err, zap.String("userID", user.ID))
	}

	user.UserServers = make([]types.IndexServer, len(indexServerRows))
	for i, row := range indexServerRows {
		user.UserServers[i] = types.IndexServer{
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

	if err := serverAssets.ResolveIndexServers(d.Context, user.UserServers); err != nil {
		return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving index server.", err)
	}

	// Get user teams
	// Teams the user is a member in
	teamIdRows, err := q.GetUserTeamIDs(d.Context, user.ID)

	if err != nil {
		return resp.Err("Error while getting user teams [db fetch]", err, zap.String("userID", user.ID))
	}

	// Ensure this always marshals as `[]` rather than `null` when the user
	// isn't on any teams — a nil Go slice serializes to JSON null, which
	// crashes frontend consumers that call .length/.map on it without a
	// null check.
	user.UserTeams = []types.Team{}

	for _, tidRaw := range teamIdRows {
		tid := uuid.UUID(tidRaw.Bytes).String()

		teamRow, err := q.GetTeamByID(d.Context, tid)

		if err != nil {
			return resp.Err("Error while getting team [db fetch]", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		var teamExtraLinks []types.Link
		if err := json.Unmarshal(teamRow.ExtraLinks, &teamExtraLinks); err != nil {
			return resp.Err("Error parsing team extra_links [json]", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		eto := types.Team{
			ID:               teamRow.ID,
			Name:             teamRow.Name,
			Short:            teamRow.Short,
			Tags:             teamRow.Tags,
			VoteBanned:       teamRow.VoteBanned,
			ApproximateVotes: int(teamRow.ApproximateVotes),
			ExtraLinks:       teamExtraLinks,
			NSFW:             teamRow.Nsfw,
			VanityRef:        teamRow.VanityRef,
			Service:          teamRow.Service,
			CreatedAt:        teamRow.CreatedAt.Time,
			UpdatedAt:        teamRow.UpdatedAt.Time,
		}

		if eto.Tags == nil {
			eto.Tags = []string{}
		}
		if eto.ExtraLinks == nil {
			eto.ExtraLinks = []types.Link{}
		}

		eto.Entities, err = resolvers.GetTeamEntities(d.Context, tid, []string{
			"team_member",
			"bot",
			"server",
		})

		if err != nil {
			return resp.Err("Error while getting team entities", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		// Votes is db:"-" (resolved in application code, not scanned from the
		// row above) — without this, every team embedded here would silently
		// report 0 votes instead of the real count that GET /teams/{id} shows.
		eto.Votes, err = votes.EntityGetVoteCount(d.Context, state.Pool, tid, "team")

		if err != nil {
			return resp.Err("Error while getting team vote count", err, zap.String("teamID", tid), zap.String("userID", user.ID))
		}

		user.UserTeams = append(user.UserTeams, eto)
	}

	// Packs
	packRows, err := q.GetUserPacksByOwner(d.Context, user.ID)

	if err != nil {
		return resp.Err("Error while getting user packs [db fetch]", err, zap.String("userID", user.ID))
	}

	user.UserPacks = make([]types.BotPack, len(packRows))
	for i, row := range packRows {
		user.UserPacks[i] = types.BotPack{
			Owner:      row.Owner,
			Name:       row.Name,
			Short:      row.Short,
			Tags:       row.Tags,
			URL:        row.Url,
			CreatedAt:  row.CreatedAt.Time,
			PackType:   row.PackType,
			Bots:       row.Bots,
			Servers:    row.Servers,
			VoteBanned: row.VoteBanned,
		}
	}

	for i := range user.UserPacks {
		err = assets.ResolveBotPack(d.Context, &user.UserPacks[i])

		if err != nil {
			return resp.ErrBody("Error while resolving user pack", "Error resolving user pack.", err, zap.String("userID", user.ID), zap.String("url", user.UserPacks[i].URL))
		}
	}

	// Fetch staff status
	positions, err := q.GetStaffPositionCount(d.Context, user.ID)

	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return resp.ErrBody("Error while getting staff status", "Error getting staff status.", err, zap.String("userID", user.ID))
	}

	user.Staff = positions > 0

	return uapi.HttpResponse{
		Json: user,
	}
}
