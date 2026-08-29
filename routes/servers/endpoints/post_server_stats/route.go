package post_server_stats

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
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

	q := db.New(tx)

	err = q.UpdateServerStatsMeta(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error while updating last_stats_post", err, zap.String("serverID", d.Auth.ID))
	}

	if payload.TotalMembers > 0 {
		err := q.UpdateServerTotalMembers(d.Context, db.UpdateServerTotalMembersParams{
			TotalMembers: int32(payload.TotalMembers),
			ServerID:     d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating total_members", err, zap.String("serverID", d.Auth.ID))
		}
	}

	if payload.OnlineMembers > 0 {
		err := q.UpdateServerOnlineMembers(d.Context, db.UpdateServerOnlineMembersParams{
			OnlineMembers: int32(payload.OnlineMembers),
			ServerID:      d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating online_members", err, zap.String("serverID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("serverID", d.Auth.ID), zap.Any("payload", payload))
	}

	return resp.NoContent()
}
