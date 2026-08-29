// Package get_server implements GET /servers/{id} — "Get Server".
//
// The target page of the request if any.
package get_server

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"popplio/api/resp"

	"github.com/PlexiOSS/Keel/uuidutil"
	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server",
		Description: "Gets a server by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The servers ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name: "target",
				Description: `The target page of the request if any.

If target is 'page', then unique clicks will be counted based on a SHA-256 hashed IP

If target is 'invite', then the invite will be counted as a click

Officially recognized targets:

- page -> server page view
- settings -> server settings page view
- invite -> server invite view`,
				Required: false,
				In:       "query",
				Schema:   docs.IdSchema,
			},
			{
				Name:        "include",
				Description: "What extra fields to include, comma-seperated.\n`long` => server long description",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "team_includes",
				Description: "What entities of the servers team to include",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func handleAnalytics(r *http.Request, id, target string) error {
	switch target {
	case "page":
		// Get IP from request and hash it
		hashedIp := fmt.Sprintf("%x", sha256.Sum256([]byte(r.RemoteAddr)))

		// Create transaction
		tx, err := state.Pool.Begin(state.Context)

		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}

		defer tx.Rollback(state.Context)

		q := db.New(tx)

		err = q.UpdateServerClicks(state.Context, id)

		if err != nil {
			return fmt.Errorf("error updating clicks count: %w", err)
		}

		// Check if the IP has already clicked the server by checking the unique_clicks row
		hasClicked, err := q.CheckServerHasUniqueClick(state.Context, db.CheckServerHasUniqueClickParams{
			HashedIp: hashedIp,
			ServerID: id,
		})

		if err != nil {
			return fmt.Errorf("error checking for any unique clicks from this user: %w", err)
		}

		if !hasClicked {
			// If not, add it to the array
			state.Logger.Debug("Adding new unique click for user during handleAnalytics", zap.String("id", id), zap.String("target", target), zap.String("targetType", "bot"))
			err = q.AppendServerUniqueClick(state.Context, db.AppendServerUniqueClickParams{
				ArrayAppend: hashedIp,
				ServerID:    id,
			})

			if err != nil {
				return fmt.Errorf("error adding new unique click for user: %w", err)
			}
		}

		// Commit transaction
		err = tx.Commit(state.Context)

		if err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
		}
	case "invite":
		// Update clicks
		err := db.New(state.Pool).UpdateServerInviteClicks(state.Context, id)

		if err != nil {
			return fmt.Errorf("error updating invite clicks: %w", err)
		}
	}

	return nil
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	target := r.URL.Query().Get("target")

	q := db.New(state.Pool)

	row, err := q.GetServerByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting server [db fetch]", err, zap.String("id", id), zap.String("target", target))
	}

	var extraLinks []types.Link
	if err := json.Unmarshal(row.ExtraLinks, &extraLinks); err != nil {
		return resp.Err("Error parsing server extra_links [json]", err, zap.String("id", id), zap.String("target", target))
	}

	server := types.Server{
		ServerID:               row.ServerID,
		Name:                   row.Name,
		Avatar:                 row.Avatar,
		TotalMembers:           int(row.TotalMembers),
		OnlineMembers:          int(row.OnlineMembers),
		Short:                  row.Short,
		Type:                   row.Type,
		Note:                   pgtype.Text{String: row.ApprovalNote, Valid: true},
		State:                  row.State,
		Tags:                   row.Tags,
		VanityRef:              row.VanityRef,
		ExtraLinks:             extraLinks,
		TeamOwnerID:            row.TeamOwner,
		InviteClicks:           int(row.InviteClicks),
		Clicks:                 int(row.Clicks),
		NSFW:                   row.Nsfw,
		ApproximateVotes:       int(row.ApproximateVotes),
		VoteBanned:             row.VoteBanned,
		Premium:                row.Premium,
		StartPeriod:            row.StartPremiumPeriod,
		PremiumPeriodLength:    row.PremiumPeriodLength,
		CaptchaOptOut:          row.CaptchaOptOut,
		CreatedAt:              row.CreatedAt,
		ClaimedBy:              row.ClaimedBy,
		LastClaimed:            row.LastClaimed,
		LoginRequiredForInvite: row.LoginRequiredForInvite,
		ShowEmojis:             row.ShowEmojis,
		Emojis:                 row.Emojis,
		Stickers:               row.Stickers,
		EmojisSyncedAt:         row.EmojisSyncedAt,
		SupporterBadge:         row.SupporterBadge,
		BoostedUntil:           row.BoostedUntil,
		FeaturedUntil:          row.FeaturedUntil,
		SpotlightedUntil:       row.SpotlightedUntil,
		VoteBlitzUntil:         row.VoteBlitzUntil,
		DiscordNSFWLevel:       int(row.DiscordNsfwLevel),
		NSFWChannelCount:       int(row.NsfwChannelCount),
		ModerationFlagged:      row.ModerationFlagged,
		ModerationCategories:   row.ModerationCategories,
	}

	teamRow, err := q.GetTeamByID(d.Context, uuidutil.Encode(server.TeamOwnerID.Bytes))

	if err != nil {
		return resp.Err("Error while getting team [db fetch]", err, zap.String("id", id), zap.String("target", target))
	}

	var teamExtraLinks []types.Link
	if err := json.Unmarshal(teamRow.ExtraLinks, &teamExtraLinks); err != nil {
		return resp.Err("Error parsing team extra_links [json]", err, zap.String("id", id), zap.String("target", target))
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

	if r.URL.Query().Get("team_includes") != "" {
		includesSplit := strings.Split(r.URL.Query().Get("team_includes"), ",")

		if len(includesSplit) > 16 {
			return resp.BadRequest("Too many `team_includes`. Maximum is 16")
		}

		eto.Entities, err = resolvers.GetTeamEntities(d.Context, eto.ID, includesSplit)

		if err != nil {
			return resp.ErrDetail("Error while getting team entities", err, zap.String("id", id), zap.String("target", target), zap.String("teamOwner", uuidutil.Encode(server.TeamOwnerID.Bytes)))
		}
	} else {
		eto.Entities = &types.TeamEntities{
			Targets: []string{}, // We don't provide any entities right now, may change
		}
	}

	server.TeamOwner = &eto

	uniqueClicks, err := q.GetServerUniqueClicksCount(d.Context, server.ServerID)

	if err != nil {
		return resp.Err("Error while getting unique clicks", err, zap.String("id", id), zap.String("target", target))
	}

	server.UniqueClicks = int64(uniqueClicks)

	code, err := q.GetVanityCodeByItag(d.Context, server.VanityRef)

	if err != nil {
		return resp.Err("Error while getting bot vanity code [db collect]", err, zap.String("id", id), zap.String("target", target), zap.String("serverID", server.ServerID))
	}

	server.Vanity = code

	// The owner may have synced emoji/sticker data sitting in the DB from
	// when show_emojis was previously on — don't leak it once they've opted
	// back out, rather than waiting for the next sync pass to clear it.
	if !server.ShowEmojis {
		server.Emojis = nil
		server.Stickers = nil
	}
	// Nil Go slices serialize to JSON null, which crashes frontend consumers
	// that call .length/.map on it without a null check.
	if server.Emojis == nil {
		server.Emojis = []types.Emoji{}
	}
	if server.Stickers == nil {
		server.Stickers = []types.Sticker{}
	}

	server.Votes, err = votes.EntityGetVoteCount(d.Context, state.Pool, server.ServerID, "server")

	if err != nil {
		return resp.ErrBody("Error while getting server vote count [db fetch]", "Error while getting server vote count [db fetch].", err)
	}

	// Handle extra includes
	if r.URL.Query().Get("include") != "" {
		includesSplit := strings.Split(r.URL.Query().Get("include"), ",")

		for _, include := range includesSplit {
			switch include {
			case "long":
				// Fetch long description
				long, err := q.GetServerLongDescription(d.Context, server.ServerID)

				if err != nil {
					return resp.Err("Error while getting bot server description [db fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("serverID", server.ServerID))
				}

				server.Long = long
			}
		}
	}

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while handling analytics", zap.Any("panic", rec), zap.String("id", id), zap.String("target", target))
			}
		}()

		if err := handleAnalytics(r, id, target); err != nil {
			state.Logger.Error("Error while handling analytics", zap.Error(err), zap.String("id", id), zap.String("target", target))
		}
	}()

	return uapi.HttpResponse{
		Json: server,
	}
}
