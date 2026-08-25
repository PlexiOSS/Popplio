package post_server_stats

import (
	"net/http"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.ServerStats{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Post Server Stats",
		Description: "This endpoint posts the stats of a server. Posting at all marks the server as self-managing its stats, which stops the periodic Infernoplex sync from overwriting total_members/online_members for it.",
		Req:         types.ServerStats{},
		Resp:        types.ApiError{},
	}
}

type statUpdate struct {
	column  string
	value   any
	present bool
}

func statUpdates(payload types.ServerStats) []statUpdate {
	return []statUpdate{
		{"total_members", payload.TotalMembers, payload.TotalMembers > 0},
		{"online_members", payload.OnlineMembers, payload.OnlineMembers > 0},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.ServerStats

	marshalResp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return marshalResp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("serverID", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	_, err = tx.Exec(d.Context, "UPDATE servers SET last_stats_post = NOW(), stats_self_managed = true WHERE server_id = $1", d.Auth.ID)

	if err != nil {
		return resp.Err("Error while updating last_stats_post", err, zap.String("serverID", d.Auth.ID))
	}

	for _, update := range statUpdates(payload) {
		if !update.present {
			continue
		}

		_, err := tx.Exec(d.Context, "UPDATE servers SET "+update.column+" = $1 WHERE server_id = $2", update.value, d.Auth.ID)

		if err != nil {
			return resp.Err("Error while updating "+update.column, err, zap.String("serverID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("serverID", d.Auth.ID), zap.Any("payload", payload))
	}

	return resp.NoContent()
}
