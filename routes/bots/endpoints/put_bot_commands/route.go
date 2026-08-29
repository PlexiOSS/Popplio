package put_bot_commands

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.UpdateBotCommands{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Bot Commands",
		Description: "Replaces the whole list of documented commands for a bot. You must have 'Edit Bot Settings' in the team if the bot is in a team.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.UpdateBotCommands{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "id")

	var payload types.UpdateBotCommands

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if len(payload.Commands) > types.MaxBotCommands {
		return resp.BadRequest("A bot can have at most 100 documented commands")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Failed to start transaction", err, zap.String("botID", botId))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	if err := q.DeleteBotCommands(d.Context, botId); err != nil {
		return resp.Err("Failed to clear existing commands", err, zap.String("botID", botId))
	}

	for i, cmd := range payload.Commands {
		err := q.InsertBotCommand(d.Context, db.InsertBotCommandParams{
			BotID:       botId,
			Name:        cmd.Name,
			Description: cmd.Description,
			Usage:       cmd.Usage,
			Category:    cmd.Category,
			Position:    int32(i),
		})

		if err != nil {
			return resp.Err("Failed to insert command", err, zap.String("botID", botId), zap.String("command", cmd.Name))
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return resp.Err("Failed to commit transaction", err, zap.String("botID", botId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
