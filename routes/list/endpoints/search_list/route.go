// Package search_list implements POST /list/search — "Search List".
//
// Searches the list returning a list of bots/servers/teams/packs that match
// the query
package search_list

import (
	"net/http"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	packAssets "popplio/routes/packs/assets"
	serverAssets "popplio/routes/servers/assets"
	teamAssets "popplio/routes/teams/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.SearchQuery{})

const (
	perPage = 12
	maxPage = 1000
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Search List",
		Description: "Searches the list returning a list of bots/servers/teams/packs that match the query",
		Req:         types.SearchQuery{},
		Resp:        types.SearchResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.SearchQuery

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if payload.Query == "" && len(payload.TagFilter.Tags) == 0 {
		// Return 206 because the user didn't specify a query or tags
		//
		// Clients can then use this to not show any bots
		return resp.Status(http.StatusPartialContent, "No query or tags specified")
	}

	// Default, if not specified
	if payload.TagFilter.TagMode == "" {
		payload.TagFilter.TagMode = types.TagModeAny
	}

	if len(payload.TagFilter.Tags) == 0 {
		payload.TagFilter.Tags = []string{}
	}

	if len(payload.TargetTypes) == 0 {
		return resp.BadRequest("No target types specified")
	}

	if payload.TagFilter.TagMode != types.TagModeAll && payload.TagFilter.TagMode != types.TagModeAny {
		return resp.BadRequest("Invalid tag mode")
	}

	if payload.Page < 1 {
		payload.Page = 1
	}

	if payload.Page > maxPage {
		return resp.BadRequest("page is too high")
	}

	sr := types.SearchResponse{Counts: map[string]int{}, PerPage: perPage}
	tagMode := string(payload.TagFilter.TagMode)
	lowerQuery := strings.ToLower(strings.TrimSpace(payload.Query))
	escaped := likeEscaper.Replace(lowerQuery)
	pattern := "%" + escaped + "%"
	prefix := escaped + "%"
	offset := int32((payload.Page - 1) * perPage)

	tags := make([]string, len(payload.TagFilter.Tags))
	for i, tag := range payload.TagFilter.Tags {
		tags[i] = strings.ToLower(strings.TrimSpace(tag))
	}
	q := db.New(state.Pool)

	for _, targetType := range payload.TargetTypes {
		switch targetType {
		case "bot":
			sr.TargetTypes = append(sr.TargetTypes, "bot")

			rows, err := q.SearchBotsPublic(d.Context, db.SearchBotsPublicParams{
				ServersFrom: int32(payload.Servers.From),
				ServersTo:   int32(payload.Servers.To),
				VotesFrom:   int32(payload.Votes.From),
				VotesTo:     int32(payload.Votes.To),
				ShardsFrom:  int32(payload.Shards.From),
				ShardsTo:    int32(payload.Shards.To),
				Tags:        tags,
				TagMode:     tagMode,
				Query:       lowerQuery,
				Pattern:     pattern,
				Prefix:      prefix,
				Lim:         perPage,
				Off:         offset,
			})

			if err != nil {
				return resp.ErrBody("Failed to query", "Error querying.", err, zap.String("targetType", "bot"))
			}

			bots := make([]types.IndexBot, len(rows))

			for i, row := range rows {
				bots[i] = types.IndexBot{
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

			if err := botAssets.ResolveIndexBots(d.Context, bots); err != nil {
				return resp.ErrBody("Error resolving bot", "Error resolving bot.", err)
			}

			sr.Bots = bots
			if len(rows) > 0 {
				sr.Counts["bot"] = int(rows[0].TotalCount)
			} else {
				sr.Counts["bot"] = 0
			}
		case "server":
			sr.TargetTypes = append(sr.TargetTypes, "server")

			rows, err := q.SearchServersPublic(d.Context, db.SearchServersPublicParams{
				MembersFrom: int32(payload.TotalMembers.From),
				MembersTo:   int32(payload.TotalMembers.To),
				VotesFrom:   int32(payload.Votes.From),
				VotesTo:     int32(payload.Votes.To),
				Tags:        tags,
				TagMode:     tagMode,
				Query:       lowerQuery,
				Pattern:     pattern,
				Prefix:      prefix,
				Lim:         perPage,
				Off:         offset,
			})

			if err != nil {
				return resp.Err("Failed to query", err, zap.String("targetType", "server"))
			}

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

			if err := serverAssets.ResolveIndexServers(d.Context, servers); err != nil {
				return resp.ErrBody("Failed to resolve server", "Error resolving server.", err)
			}

			sr.Servers = servers
			if len(rows) > 0 {
				sr.Counts["server"] = int(rows[0].TotalCount)
			} else {
				sr.Counts["server"] = 0
			}
		case "team":
			sr.TargetTypes = append(sr.TargetTypes, "team")

			rows, err := q.SearchTeamsPublic(d.Context, db.SearchTeamsPublicParams{
				Query:     lowerQuery,
				Pattern:   pattern,
				Prefix:    prefix,
				Tags:      tags,
				TagMode:   tagMode,
				Lim:       perPage,
				Off:       offset,
				VotesFrom: int32(payload.Votes.From),
				VotesTo:   int32(payload.Votes.To),
			})

			if err != nil {
				return resp.Err("Failed to query", err, zap.String("targetType", "team"))
			}

			teams := make([]types.Team, len(rows))

			for i, row := range rows {
				teams[i] = types.Team{
					ID:               row.ID,
					Name:             row.Name,
					Short:            pgtype.Text{String: row.Short, Valid: true},
					Tags:             row.Tags,
					NSFW:             row.Nsfw,
					VoteBanned:       row.VoteBanned,
					ApproximateVotes: int(row.ApproximateVotes),
				}
			}

			teamAssets.ResolveIndexTeams(teams)

			if err := teamAssets.ResolveTeamAssetVersions(d.Context, teams); err != nil {
				return resp.Err("Failed to resolve team asset versions", err, zap.String("targetType", "team"))
			}

			sr.Teams = teams
			if len(rows) > 0 {
				sr.Counts["team"] = int(rows[0].TotalCount)
			} else {
				sr.Counts["team"] = 0
			}
		case "pack":
			sr.TargetTypes = append(sr.TargetTypes, "pack")

			rows, err := q.SearchPacksPublic(d.Context, db.SearchPacksPublicParams{
				Query:   lowerQuery,
				Pattern: pattern,
				Prefix:  prefix,
				Tags:    tags,
				TagMode: tagMode,
				Lim:     perPage,
				Off:     offset,
			})

			if err != nil {
				return resp.Err("Failed to query", err, zap.String("targetType", "pack"))
			}

			packs := make([]types.BotPack, len(rows))

			for i, row := range rows {
				packs[i] = types.BotPack{
					Owner:      row.Owner,
					Name:       row.Name,
					Short:      row.Short,
					URL:        row.Url,
					PackType:   row.PackType,
					Tags:       row.Tags,
					Bots:       row.Bots,
					Servers:    row.Servers,
					VoteBanned: row.VoteBanned,
				}

				if err := packAssets.ResolveBotPack(d.Context, &packs[i]); err != nil {
					return resp.ErrBody("Failed to resolve pack", "Error resolving pack.", err)
				}
			}

			sr.Packs = packs
			if len(rows) > 0 {
				sr.Counts["pack"] = int(rows[0].TotalCount)
			} else {
				sr.Counts["pack"] = 0
			}
		}
	}

	if len(sr.TargetTypes) == 0 {
		sr.TargetTypes = []string{}
	}

	return uapi.HttpResponse{
		Json: sr,
	}
}
