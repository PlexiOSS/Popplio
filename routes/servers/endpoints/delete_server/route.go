// Package delete_server implements DELETE /servers/{id} — "Delete Server".
//
// Deletes a server from the list. This is *irreversible*. You must have
// 'Delete Servers' in the server's team. Returns 204 on success.
package delete_server

import (
	"fmt"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Server",
		Description: "Deletes a server from the list. This is *irreversible*. You must have 'Delete Servers' in the server's team. Returns 204 on success.",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID), zap.String("serverID", id))
	}

	defer tx.Rollback(d.Context)

	err = db.New(tx).DeleteServerByID(d.Context, id)

	if err != nil {
		return resp.Err("Error while deleting server", err, zap.String("userID", d.Auth.ID), zap.String("serverID", id))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID), zap.String("serverID", id))
	}

	// Send embed to bot log channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, discord.MessageCreate{
		Content: "",
		Embeds: []discord.Embed{
			{
				URL:   state.Config.Sites.Frontend + "/servers/" + id,
				Title: "Server Deleted",
				Color: 0xff0000,
				Fields: []discord.EmbedField{
					{
						Name:  "Server ID",
						Value: id,
					},
					{
						Name:  "Deleter",
						Value: fmt.Sprintf("<@%s>", d.Auth.ID),
					},
				},
			},
		},
	})

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusOK,
			Data:   "Server deleted, but the mod log message couldn't be sent",
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
