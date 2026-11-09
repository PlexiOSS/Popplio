// Package get_bot implements GET /bots/{id} — "Get Bot".
//
// The target page of the request if any.
package get_bot

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"popplio/api"
	"popplio/api/resp"
	"popplio/botpresence"
	"popplio/entityassets"
	"popplio/listing"

	"github.com/PlexiOSS/Keel/uuidutil"
	"popplio/db"
	botassets "popplio/routes/bots/assets"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/votes"

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
		Summary:     "Get Bot",
		Description: "Gets a bot by id",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bots ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target",
				Description: "No longer used. Views and invite clicks are recorded with POST /bots/{id}/views and POST /bots/{id}/invite-clicks.",
				Required: false,
				In:       "query",
				Schema:   docs.IdSchema,
			},
			{
				Name:        "include",
				Description: "What extra fields to include, comma-seperated.`long` => bot long description",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "team_includes",
				Description: "If the bot is team-owned, what entities of the team to include",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Bot{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	target := r.URL.Query().Get("target")

	q := db.New(state.Pool)

	row, err := q.GetBotByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.NotFound("No bots could be found matching your query")

	}

	if err != nil {
		return resp.ErrDetail("Error while getting bot [db fetch]", err, zap.String("id", id), zap.String("target", target))
	}

	unlisted := !listing.IsPublic(row.Type)

	if unlisted && !listing.CanViewUnlisted(d.Context, d.Auth, api.TargetTypeBot, row.BotID) {
		return resp.NotFound("No bots could be found matching your query")
	}

	var extraLinks []types.Link
	if err := json.Unmarshal(row.ExtraLinks, &extraLinks); err != nil {
		return resp.ErrDetail("Error parsing bot extra_links [json]", err, zap.String("id", id), zap.String("target", target))
	}

	shardList := make([]int, len(row.ShardList))
	for i, s := range row.ShardList {
		shardList[i] = int(s)
	}

	uptimePercent, uptimeChecks := botassets.RecentUptime(d.Context, row.BotID)

	bot := types.Bot{
		ITag:                 row.Itag,
		BotID:                row.BotID,
		ClientID:             row.ClientID,
		ExtraLinks:           extraLinks,
		Tags:                 row.Tags,
		Prefix:               row.Prefix,
		Owner:                row.Owner,
		Short:                row.Short,
		Library:              row.Library,
		NSFW:                 row.Nsfw,
		Premium:              row.Premium,
		LastStatsPost:        row.LastStatsPost,
		LastJapiUpdate:       row.LastJapiUpdate,
		Servers:              int(row.Servers),
		Shards:               int(row.Shards),
		ShardList:            shardList,
		Users:                int(row.Users),
		ApproximateVotes:     int(row.ApproximateVotes),
		Clicks:               int(row.Clicks),
		InviteClicks:         int(row.InviteClicks),
		Invite:               row.Invite,
		Type:                 row.Type,
		VanityRef:            row.VanityRef,
		VoteBanned:           row.VoteBanned,
		StartPeriod:          row.StartPremiumPeriod,
		PremiumPeriodLength:  row.PremiumPeriodLength,
		CertReason:           row.CertReason,
		Uptime:               int(row.Uptime),
		TotalUptime:          int(row.TotalUptime),
		HiddenStats:          row.HiddenStats,
		UptimePercent:        uptimePercent,
		UptimeChecks:         uptimeChecks,
		UptimeLastChecked:    row.UptimeLastChecked,
		Note:                 pgtype.Text{String: row.ApprovalNote, Valid: true},
		CreatedAt:            row.CreatedAt,
		ClaimedBy:            row.ClaimedBy,
		UpdatedAt:            row.UpdatedAt,
		LastClaimed:          row.LastClaimed,
		TeamOwnerID:          row.TeamOwner,
		CaptchaOptOut:        row.CaptchaOptOut,
		AutoVotesOptOut:      row.AutoVotesOptOut,
		HighlightedUntil:     row.HighlightedUntil,
		CardTheme:            row.CardTheme,
		Highlighted:          row.HighlightedUntil.Valid && row.HighlightedUntil.Time.After(time.Now()),
		SelfStatus:           row.SelfStatus,
		SupporterBadge:       row.SupporterBadge,
		BoostedUntil:         row.BoostedUntil,
		FeaturedUntil:        row.FeaturedUntil,
		SpotlightedUntil:     row.SpotlightedUntil,
		VoteBlitzUntil:       row.VoteBlitzUntil,
		ModerationFlagged:    row.ModerationFlagged,
		ModerationCategories: row.ModerationCategories,
	}

	if bot.Owner.Valid {
		ownerUser, err := dovewing.GetUser(d.Context, bot.Owner.String, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.ErrBody("Error while getting bot owner [dovewing fetch]", "Error while getting bot [dovewing fetch].", err, zap.String("id", id), zap.String("target", target), zap.String("owner", bot.Owner.String))
		}

		bot.MainOwner = ownerUser
	} else {
		teamRow, err := q.GetTeamByID(d.Context, uuidutil.Encode(bot.TeamOwnerID.Bytes))

		if err != nil {
			return resp.ErrDetail("Error while getting bot team owner [db fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("teamOwner", uuidutil.Encode(bot.TeamOwnerID.Bytes)))
		}

		var teamExtraLinks []types.Link
		if err := json.Unmarshal(teamRow.ExtraLinks, &teamExtraLinks); err != nil {
			return resp.ErrDetail("Error parsing team extra_links [json]", err, zap.String("id", id), zap.String("target", target))
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

		eto.AssetVersions, err = entityassets.Get(d.Context, api.TargetTypeTeam, eto.ID)

		if err != nil {
			return resp.ErrDetail("Error while getting team asset versions [db fetch]", err, zap.String("id", id), zap.String("target", target))
		}

		if r.URL.Query().Get("team_includes") != "" {
			includesSplit := strings.Split(r.URL.Query().Get("team_includes"), ",")

			if len(includesSplit) > 16 {
				return resp.BadRequest("Too many `team_includes`. Maximum is 16")
			}

			eto.Entities, err = resolvers.GetTeamEntities(d.Context, eto.ID, includesSplit, listing.CanViewUnlisted(d.Context, d.Auth, api.TargetTypeTeam, eto.ID))

			if err != nil {
				return resp.ErrDetail("Error while getting team entities", err, zap.String("id", id), zap.String("target", target), zap.String("teamOwner", uuidutil.Encode(bot.TeamOwnerID.Bytes)))
			}
		} else {
			eto.Entities = &types.TeamEntities{
				Targets: []string{}, // We don't provide any entities right now, may change
			}
		}

		bot.TeamOwner = &eto
	}

	botUser, err := dovewing.GetUser(d.Context, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.ErrDetail("Error while getting bot user [dovewing fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
	}

	bot.User = botUser
	botassets.ApplySelfStatus(bot.User, bot.SelfStatus.String, bot.Servers, bot.LastStatsPost, botpresence.Get(d.Context, bot.BotID), row.LastSeenOnline)

	uniqueClicks, err := q.GetBotUniqueClicksCount(d.Context, bot.BotID)

	if err != nil {
		return resp.ErrDetail("Error while getting bot unique clicks [db fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
	}

	bot.UniqueClicks = int64(uniqueClicks)

	bot.AssetVersions, err = entityassets.Get(d.Context, api.TargetTypeBot, bot.BotID)

	if err != nil {
		return resp.ErrDetail("Error while getting bot asset versions [db fetch]", err, zap.String("id", id), zap.String("target", target))
	}

	code, err := q.GetVanityCodeByItag(d.Context, bot.VanityRef)

	if err != nil {
		return resp.ErrDetail("Error while getting bot vanity code [db fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
	}

	bot.Vanity = code

	if row.CardTheme {
		bot.CardColor = row.CardColor.String
	}

	bot.Votes, bot.AutoVotes, err = votes.EntityGetVoteCounts(d.Context, state.Pool, bot.BotID, "bot")

	if err != nil {
		return resp.ErrBody("Error while getting bot vote count [db fetch]", "Error while getting bot vote count [db fetch].", err)
	}

	recentViews, err := q.GetEntityRecentViews(d.Context, db.GetEntityRecentViewsParams{TargetType: "bot", TargetID: bot.BotID})

	if err != nil {
		return resp.Err("Error while getting recent views", err)
	}

	bot.Views30d = int(recentViews)

	if len(bot.HiddenStats) > 0 && !listing.CanViewUnlisted(d.Context, d.Auth, api.TargetTypeBot, bot.BotID) {
		listing.HideBotStats(&bot)
	}

	// Handle extra includes
	if r.URL.Query().Get("include") != "" {
		includesSplit := strings.Split(r.URL.Query().Get("include"), ",")

		for _, include := range includesSplit {
			switch include {
			case "long":
				// Fetch long description
				long, err := q.GetBotLongDescription(d.Context, bot.BotID)

				if err != nil {
					return resp.ErrDetail("Error while getting bot long description [db fetch]", err, zap.String("id", id), zap.String("target", target), zap.String("botID", bot.BotID))
				}

				bot.Long = long
			}
		}
	}

	return uapi.HttpResponse{
		Json: bot,
	}
}
