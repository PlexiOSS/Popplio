package patch_bot_settings

import (
	"encoding/json"
	"fmt"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/PlexiOSS/Keel/ptr"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.BotSettingsUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Bot Settings",
		Description: "Updates a bots settings. You must have 'Edit Bot Settings' in the team if the bot is in a team. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.BotSettingsUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	var payload types.BotSettingsUpdate

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
		return resp.Err("Error marshaling extra links", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	err = db.New(state.Pool).UpdateBotSettings(d.Context, db.UpdateBotSettingsParams{
		Short:         payload.Short,
		Long:          payload.Long,
		Prefix:        payload.Prefix,
		Invite:        payload.Invite,
		Library:       payload.Library,
		ExtraLinks:    extraLinksJSON,
		Tags:          payload.Tags,
		Nsfw:          payload.NSFW,
		CaptchaOptOut: payload.CaptchaOptOut,
		BotID:         id,
	})

	if err != nil {
		return resp.Err("Failed to update bot: ", err, zap.String("userID", d.Auth.ID), zap.String("botID", id))
	}

	botUser, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		state.Logger.Error("Failed to get bot user for update embed", zap.Error(err), zap.String("userID", d.Auth.ID), zap.String("botID", id))
		botUser = nil
	}

	botName := "<@" + id + ">"

	if botUser != nil && botUser.Username != "" {
		botName = botUser.Username
	}

	embed := discord.Embed{
		URL:   state.Config.Sites.Frontend + "/bots/" + id,
		Title: "Bot Updated",
		Fields: []discord.EmbedField{
			{
				Name:   "Name",
				Value:  botName,
				Inline: ptr.TruePtr,
			},
			{
				Name:   "Bot ID",
				Value:  "<@" + id + ">",
				Inline: ptr.TruePtr,
			},
			{
				Name:   "User",
				Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
				Inline: ptr.TruePtr,
			},
		},
	}

	if botUser != nil && botUser.Avatar != "" {
		embed.Thumbnail = &discord.EmbedResource{URL: botUser.Avatar}
	}

	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
		Content: "",
		Embeds:  []discord.Embed{embed},
	})

	if err != nil {
		state.Logger.Error("Error while sending update embed to mod logs channel", zap.Error(err), zap.String("botID", id))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
