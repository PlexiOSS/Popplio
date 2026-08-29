package patch_pack

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var compiledMessages = uapi.CompileValidationErrors(PatchPack{})

type PatchPack struct {
	Name    string                 `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	Short   string                 `json:"short" validate:"required,min=10,max=100,noxss" msg:"Description must be between 10 and 100 characters"`
	Tags    []string               `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	Bots    []string               `json:"bots" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 bots without duplicates"`
	Servers []string               `json:"servers" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 servers without duplicates"`
	Emojis  []types.PackEmojiInput `json:"emojis" validate:"omitempty,max=50,dive" msg:"There can be at most 50 emojis"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch Pack",
		Description: "Edits a pack you are owner of based on the URL only. Returns 204 on success",
		Req:         PatchPack{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "id",
				Description: "The pack's URL",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload PatchPack

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	var id = chi.URLParam(r, "id")

	q := db.New(state.Pool)

	ownerType, err := q.GetPackOwnerAndType(d.Context, id)
	owner, packType := ownerType.Owner, ownerType.PackType

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while checking pack owner [db fetch]", err, zap.String("id", id))
	}

	if owner != d.Auth.ID {
		return resp.Forbidden("You are not the owner of this pack")
	}

	switch packType {
	case types.PackTypeBot:
		if len(payload.Bots) == 0 {
			return resp.BadRequest("A bot pack must contain at least one bot")
		}
	case types.PackTypeServer:
		if len(payload.Servers) == 0 {
			return resp.BadRequest("A server pack must contain at least one server")
		}
	case types.PackTypeEmoji:
		if len(payload.Emojis) == 0 {
			return resp.BadRequest("An emoji pack must contain at least one emoji")
		}
	}

	for _, bot := range payload.Bots {
		botUser, err := dovewing.GetUser(d.Context, bot, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.BadRequest("One of the bot you wish to add does not exist [" + bot + "]: " + err.Error())
		}

		if !botUser.Bot {
			return resp.BadRequest("One of the bot you wish to add is not actually a bot [" + bot + "]")
		}
	}

	for _, server := range payload.Servers {
		serverCount, err := q.CountServerByID(d.Context, server)

		if err != nil {
			return resp.ErrBody("Error checking if server exists:", "Error checking if server exists: "+err.Error(), err)
		}

		if serverCount == 0 {
			return resp.BadRequest("One of the servers you wish to add does not exist [" + server + "]")
		}
	}

	if payload.Bots == nil {
		payload.Bots = []string{}
	}
	if payload.Servers == nil {
		payload.Servers = []string{}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.ErrBody("Failed to create transaction [patch_pack]", "Failed to create transaction.", err)
	}

	defer tx.Rollback(d.Context)

	txQ := db.New(tx)

	err = txQ.UpdatePack(d.Context, db.UpdatePackParams{
		Name:    payload.Name,
		Short:   payload.Short,
		Tags:    payload.Tags,
		Bots:    payload.Bots,
		Servers: payload.Servers,
		Url:     id,
	})

	if err != nil {
		return resp.Err("Error while updating pack [db exec]", err, zap.String("id", id))
	}

	if packType == types.PackTypeEmoji {
		err = txQ.DeletePackEmojis(d.Context, id)

		if err != nil {
			return resp.Err("Error while clearing existing pack emojis [db exec]", err, zap.String("id", id))
		}

		for i, emoji := range payload.Emojis {
			err = txQ.InsertPackEmoji(
				d.Context,
				db.InsertPackEmojiParams{
					ID:       emoji.ID,
					PackUrl:  id,
					Name:     emoji.Name,
					Animated: emoji.Animated,
					Position: int32(i),
				},
			)

			if err != nil {
				return resp.ErrBody("Failed to insert pack emoji [patch_pack]", "Failed to save one of the pack's emojis — the uploaded image may not exist yet.", err, zap.String("emojiId", emoji.ID))
			}
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Failed to commit transaction [patch_pack]", err, zap.String("id", id))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
