package patch_server_settings

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/PlexiOSS/Keel/ptr"
	"popplio/api/resp"
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

func updateServerArgs(server types.ServerSettingsUpdate) []any {
	return []any{
		server.Short,
		server.Long,
		server.ExtraLinks,
		server.State,
		server.Tags,
		server.NSFW,
		server.CaptchaOptOut,
		server.LoginRequiredForInvite,
		server.ShowEmojis,
	}
}

var (
	compiledMessages = uapi.CompileValidationErrors(types.ServerSettingsUpdate{})
	updateSql        = []string{}
	updateSqlStr     string
)

func Setup() {
	for i, field := range reflect.VisibleFields(reflect.TypeOf(types.ServerSettingsUpdate{})) {
		if field.Tag.Get("db") != "" {
			updateSql = append(updateSql, field.Tag.Get("db")+"=$"+strconv.Itoa(i+1))
		}
	}

	updateSqlStr = strings.Join(updateSql, ",")
}

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

	serverArgs := updateServerArgs(payload)

	if len(updateSql) != len(serverArgs) {
		return resp.ErrBody("Internal Error: The number of columns and arguments do not match", "Internal Error: The number of columns and arguments do not match", nil)
	}

	serverArgs = append(serverArgs, id)

	_, err = state.Pool.Exec(d.Context, "UPDATE servers SET "+updateSqlStr+" WHERE server_id=$"+strconv.Itoa(len(serverArgs)), serverArgs...)

	if err != nil {
		return resp.Err("Error while updating server", err, zap.String("serverID", id))
	}

	var name, avatar string

	err = state.Pool.QueryRow(d.Context, "SELECT name, avatar FROM servers WHERE server_id = $1", id).Scan(&name, &avatar)

	if err != nil {
		return resp.Err("Error while getting server info", err, zap.String("serverID", id))
	}

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
