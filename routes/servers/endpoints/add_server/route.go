package add_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"popplio/api/resp"

	"popplio/db"
	"popplio/moderation"
	"popplio/perms"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	"github.com/PlexiOSS/Keel/ptr"

	"github.com/disgoorg/disgo/discord"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/crypto"
	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.CreateServer{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Add Server",
		Description: "Adds a server to the database from an invite link. Resolves the guild via the invite (the tracking bot does not need to already be in the server). Returns 204 on success",
		Req:         types.CreateServer{},
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 5,
		Bucket:      "add_server",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error calculating ratelimits", err, zap.String("userID", d.Auth.ID))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var payload types.CreateServer

	hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	err = validators.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	invite, err := assets.ResolveInvite(d.Context, payload.Invite)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	payload.ServerID = invite.Guild.ID.String()
	payload.Name = invite.Guild.Name
	payload.Avatar = assets.GuildIconURL(invite.Guild)
	payload.TotalMembers = invite.ApproximateMemberCount
	payload.OnlineMembers = invite.ApproximatePresenceCount

	q := db.New(state.Pool)

	count, err := q.CountServerByID(d.Context, payload.ServerID)

	if err != nil {
		return resp.Err("Error while checking if server is already in database", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	if count > 0 {
		return resp.Conflict("This server is already in the database")
	}

	vanity := strings.ReplaceAll(strings.ToLower(payload.Name), " ", "-")
	vanity = regexp.MustCompile("[^a-zA-Z0-9-]").ReplaceAllString(vanity, "")
	vanity = strings.TrimSuffix(vanity, "-")

	vanityCount, err := q.CountVanityByCode(d.Context, vanity)

	if err != nil {
		return resp.Err("Error while checking if calculated vanity is already taken", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
	}

	if vanityCount > 0 {
		vanity = vanity + "-" + crypto.RandString(8)
	}

	systems, err := validators.GetWordBlacklistSystems(d.Context, vanity)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return resp.BadRequest("Error while getting word blacklist systems: " + err.Error())
	}

	if slices.Contains(systems, "vanity.code") {
		return resp.BadRequest("The chosen vanity is blacklisted")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	defer tx.Rollback(d.Context)

	txq := db.New(tx)

	if payload.TeamOwner != "" {
		entityPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", payload.TeamOwner)

		if err != nil {
			state.Logger.Error("Error while getting team perms", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("teamID", payload.TeamOwner), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
			return resp.BadRequest("Error getting user perms: " + err.Error())
		}

		if !entityPerms.Has(perms.EntityAddServers) {
			return resp.Forbidden("You do not have permission to add new servers to this team")
		}
	} else {
		var teamId = uuid.New()

		vanityRef, err := txq.InsertVanityReturningItag(d.Context, db.InsertVanityReturningItagParams{
			Code:       payload.Name + crypto.RandString(16),
			TargetID:   teamId.String(),
			TargetType: "team",
		})

		if err != nil {
			return resp.BadRequest("Error while creating vanity: " + err.Error())
		}

		err = txq.InsertTeamForAddServer(d.Context, db.InsertTeamForAddServerParams{
			ID:        teamId.String(),
			Name:      payload.Name,
			VanityRef: vanityRef,
		})

		if err != nil {
			return resp.BadRequest("Error while creating team: " + err.Error())
		}

		err = txq.InsertTeamMemberForAddServer(d.Context, db.InsertTeamMemberForAddServerParams{
			TeamID: pgtype.UUID{Bytes: teamId, Valid: true},
			UserID: d.Auth.ID,
			Flags:  []string{string(perms.EntityOwner)},
		})

		if err != nil {
			return resp.BadRequest("Error while adding team member: " + err.Error())
		}

		payload.TeamOwner = teamId.String()
	}

	itag, err := txq.InsertVanityReturningItag(d.Context, db.InsertVanityReturningItagParams{
		Code:       vanity,
		TargetID:   payload.ServerID,
		TargetType: "server",
	})

	if err != nil {
		return resp.Err("Error while inserting vanity", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID), zap.String("vanity", vanity))
	}

	payload.VanityRef = itag

	extraLinksJSON, err := json.Marshal(payload.ExtraLinks)

	if err != nil {
		return resp.Err("Error marshaling extra links", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	var teamOwnerUUID pgtype.UUID
	if err := teamOwnerUUID.Scan(payload.TeamOwner); err != nil {
		return resp.BadRequest("Invalid team ID: " + err.Error())
	}

	err = txq.InsertServer(d.Context, db.InsertServerParams{
		Invite:        payload.Invite,
		Short:         payload.Short,
		Long:          payload.Long,
		ExtraLinks:    extraLinksJSON,
		Tags:          payload.Tags,
		Nsfw:          payload.NSFW,
		TeamOwner:     teamOwnerUUID,
		ServerID:      payload.ServerID,
		Name:          payload.Name,
		Avatar:        payload.Avatar,
		TotalMembers:  int32(payload.TotalMembers),
		OnlineMembers: int32(payload.OnlineMembers),
		VanityRef:     payload.VanityRef,
	})

	if err != nil {
		return resp.Err("Error while inserting server", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	if result, err := moderation.CheckText(d.Context, payload.Short, payload.Long); err != nil {
		state.Logger.Error("Failed to run moderation check on new server", zap.Error(err), zap.String("serverID", payload.ServerID))
	} else if result := moderation.EffectiveResult(result, payload.NSFW); result.Flagged {
		if err := q.UpdateServerModerationResult(d.Context, db.UpdateServerModerationResultParams{
			ServerID:             payload.ServerID,
			ModerationFlagged:    result.Flagged,
			ModerationCategories: result.Categories,
		}); err != nil {
			state.Logger.Error("Failed to store moderation result for new server", zap.Error(err), zap.String("serverID", payload.ServerID))
		}

		if err := moderation.FileAutoReport(d.Context, "server", payload.ServerID, result.Categories); err != nil {
			state.Logger.Error("Failed to auto-file report for flagged server", zap.Error(err), zap.String("serverID", payload.ServerID))
		}
	}

	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend + "/servers/" + payload.ServerID,
				Title: "New Server Added",
				Fields: []discord.EmbedField{
					{
						Name:   "Name",
						Value:  payload.Name,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Server ID",
						Value:  payload.ServerID,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Added by",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: ptr.TruePtr,
					},
				},
			},
		},
	})

	if err != nil {
		state.Logger.Error("Error while sending server logs message", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("serverID", payload.ServerID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
