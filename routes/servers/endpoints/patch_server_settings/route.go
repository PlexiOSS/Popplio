package patch_server_settings

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PlexiOSS/Keel/ptr"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.ServerSettingsUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Server Settings",
		Description: "Updates a servers settings. You must have 'Edit Server Settings' in the team if the bot is in a team. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.ServerSettingsUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	var payload types.ServerSettingsUpdate

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	err = validators.ValidateExtraLinks(payload.ExtraLinks)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	extraLinksJSON, err := json.Marshal(payload.ExtraLinks)

	if err != nil {
		return resp.Err("Error marshaling extra links", err, zap.String("serverID", id))
	}

	q := db.New(state.Pool)

	err = q.UpdateServerSettings(d.Context, db.UpdateServerSettingsParams{
		Short:                  payload.Short,
		Long:                   payload.Long,
		ExtraLinks:             extraLinksJSON,
		State:                  payload.State,
		Tags:                   payload.Tags,
		Nsfw:                   payload.NSFW,
		CaptchaOptOut:          payload.CaptchaOptOut,
		LoginRequiredForInvite: payload.LoginRequiredForInvite,
		ShowEmojis:             payload.ShowEmojis,
		ServerID:               id,
	})

	if err != nil {
		return resp.Err("Error while updating server", err, zap.String("serverID", id))
	}

	nameAvatar, err := q.GetServerNameAndAvatar(d.Context, id)

	if err != nil {
		return resp.Err("Error while getting server info", err, zap.String("serverID", id))
	}

	name, avatar := nameAvatar.Name, nameAvatar.Avatar

	embed := discord.Embed{
		URL:   state.Config.Sites.Frontend + "/servers/" + id,
		Title: "Server Updated",
		Fields: []discord.EmbedField{
			{
				Name:   "Name",
				Value:  name,
				Inline: ptr.TruePtr,
			},
			{
				Name:   "Server ID",
				Value:  id,
				Inline: ptr.TruePtr,
			},
			{
				Name:   "User",
				Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
				Inline: ptr.TruePtr,
			},
		},
	}

	if avatar != "" {
		embed.Thumbnail = &discord.EmbedResource{URL: avatar}
	}

	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: "",
		Embeds:  []discord.Embed{embed},
	})

	if err != nil {
		state.Logger.Error("Error while sending update embed to mod logs channel", zap.Error(err), zap.String("serverID", id))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
