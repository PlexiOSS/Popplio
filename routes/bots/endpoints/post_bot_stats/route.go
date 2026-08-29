package post_bot_stats

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.BotStats{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Post Bot Stats",
		Description: "This endpoint posts the stats of a bot. `status` is optional and self-reports the bot's presence (online/idle/dnd/offline) — post it periodically to keep it fresh, it will otherwise keep showing the last value posted.",
		Req:         types.BotStats{},
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.BotStats

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
		return resp.Err("Error while starting transaction", err, zap.String("botID", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	err = q.UpdateBotLastStatsPost(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error while updating last_stats_post", err, zap.String("botID", d.Auth.ID))
	}

	if payload.Servers > 0 {
		err := q.UpdateBotServers(d.Context, db.UpdateBotServersParams{
			Servers: int32(payload.Servers),
			BotID:   d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating servers", err, zap.String("botID", d.Auth.ID))
		}
	}

	if payload.Shards > 0 {
		err := q.UpdateBotShards(d.Context, db.UpdateBotShardsParams{
			Shards: int32(payload.Shards),
			BotID:  d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating shards", err, zap.String("botID", d.Auth.ID))
		}
	}

	if payload.Users > 0 {
		err := q.UpdateBotUsers(d.Context, db.UpdateBotUsersParams{
			Users: int32(payload.Users),
			BotID: d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating users", err, zap.String("botID", d.Auth.ID))
		}
	}

	if len(payload.ShardList) > 0 {
		shardList := make([]int64, len(payload.ShardList))
		for i, s := range payload.ShardList {
			shardList[i] = int64(s)
		}

		err := q.UpdateBotShardList(d.Context, db.UpdateBotShardListParams{
			ShardList: shardList,
			BotID:     d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating shard_list", err, zap.String("botID", d.Auth.ID))
		}
	}

	if payload.Status != "" {
		err := q.UpdateBotSelfStatus(d.Context, db.UpdateBotSelfStatusParams{
			SelfStatus: pgtype.Text{String: payload.Status, Valid: true},
			BotID:      d.Auth.ID,
		})

		if err != nil {
			return resp.Err("Error while updating self_status", err, zap.String("botID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("botID", d.Auth.ID), zap.Any("payload", payload))
	}

	return resp.NoContent()
}
