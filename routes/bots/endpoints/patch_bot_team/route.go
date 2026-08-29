package patch_bot_team

import (
	"fmt"
	"net/http"

	"popplio/api"
	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/ptr"
	"github.com/PlexiOSS/Keel/uuidutil"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary: "Patch Bot Team",
		Description: `Transfers a bot to another team. 

Semantically equivalent to:
- Remove bot in question from list
- Readd bot to list with same data
- Transfer bot ownership to team

The below are the requirements for this due to the above:

- The user must have the "Delete Bots" permission in the team they are transferring the bot from
- The user must have the "Add New Bots" permission in the team they are transferring the bot to

The bots ownership will be transferred to the new team.

Returns a 204 on success`,
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bid",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.PatchBotTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "bid")

	var payload types.PatchBotTeam

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if payload.TeamID == "" {
		return resp.BadRequest("Team ID must be provided")
	}

	err := api.AuthzEntityPermissionCheck(
		d.Context,
		d.Auth,
		api.TargetTypeBot,
		id,
		perms.EntityDeleteBots,
	)

	if err != nil {
		return resp.Forbidden("You must be able to delete the bot in the old team to transfer it: " + err.Error())
	}

	err = api.AuthzEntityPermissionCheck(
		d.Context,
		d.Auth,
		api.TargetTypeTeam,
		payload.TeamID,
		perms.EntityAddBots,
	)

	if err != nil {
		return resp.Forbidden("You must be able to add the bot in the new team to transfer it: " + err.Error())
	}

	q := db.New(state.Pool)

	currentBotTeam, err := q.GetBotTeamOwner(d.Context, id)

	if err != nil {
		return resp.Err("Error getting current team for bot: ", err, zap.String("botID", id), zap.String("userID", d.Auth.ID))
	}

	var newTeamOwner pgtype.UUID
	if err := newTeamOwner.Scan(payload.TeamID); err != nil {
		return resp.BadRequest("Invalid team ID: " + err.Error())
	}

	err = q.UpdateBotTeamOwner(d.Context, db.UpdateBotTeamOwnerParams{
		TeamOwner: newTeamOwner,
		BotID:     id,
	})

	if err != nil {
		return resp.Err("Error transferring bot to team", nil, zap.String("botID", id), zap.String("userID", d.Auth.ID), zap.String("newTeamID", payload.TeamID))
	}

	state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend + "/bots/" + id,
				Title: "Bot Team Update!",
				Fields: []discord.EmbedField{
					{
						Name:   "Bot ID",
						Value:  id,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Performed By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: ptr.TruePtr,
					},
					{
						Name:  "Old Team",
						Value: fmt.Sprintf("[View Team](%s/teams/%s)", state.Config.Sites.Frontend, uuidutil.Encode(currentBotTeam.Bytes)),
					},
					{
						Name:  "New Team",
						Value: fmt.Sprintf("[View Team](%s/teams/%s)", state.Config.Sites.Frontend, payload.TeamID),
					},
				},
			},
		},
	})

	return uapi.DefaultResponse(http.StatusNoContent)
}
