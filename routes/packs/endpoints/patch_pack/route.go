// Package patch_pack implements PATCH /users/{uid}/packs/{id} — "Patch
// Pack".
//
// Edits a pack you are owner of based on the URL only. Returns 204 on
// success
package patch_pack

import (
	"errors"
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var compiledMessages = uapi.CompileValidationErrors(PatchPack{})

// PatchPack intentionally has no pack_type field — a pack's type is
// immutable after creation (see types.BotPack.PackType doc comment), so
// there is simply nothing here to change it with.
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

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	var id = chi.URLParam(r, "id")

	// Check that the pack exists and get its owner + type in one query
	var owner, packType string

	err = state.Pool.QueryRow(d.Context, "SELECT owner, pack_type FROM packs WHERE url = $1", id).Scan(&owner, &packType)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while checking pack owner [db fetch]", err, zap.String("id", id))
	}

	if owner != d.Auth.ID {
		return resp.Forbidden("You are not the owner of this pack")
	}

	// Content requirement depends on the pack's (immutable) type.
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

	// Check that all bots exist. Anyone may add any existing bot/server to a
	// pack — packs are curated lists, not something scoped to what the
	// author owns.
	for _, bot := range payload.Bots {
		botUser, err := dovewing.GetUser(d.Context, bot, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.BadRequest("One of the bot you wish to add does not exist [" + bot + "]: " + err.Error())
		}

		if !botUser.Bot {
			return resp.BadRequest("One of the bot you wish to add is not actually a bot [" + bot + "]")
		}
	}

	// Check that all servers exist
	for _, server := range payload.Servers {
		var serverCount int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", server).Scan(&serverCount)

		if err != nil {
			return resp.ErrBody("Error checking if server exists:", "Error checking if server exists: "+err.Error(), err)
		}

		if serverCount == 0 {
			return resp.BadRequest("One of the servers you wish to add does not exist [" + server + "]")
		}
	}

	// Both columns are NOT NULL — a nil Go slice encodes as SQL NULL, so
	// normalize an omitted field to an empty slice before it ever reaches a query.
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

	// Update the pack
	_, err = tx.Exec(d.Context, "UPDATE packs SET name = $1, short = $2, tags = $3, bots = $4, servers = $5 WHERE url = $6", payload.Name, payload.Short, payload.Tags, payload.Bots, payload.Servers, id)

	if err != nil {
		return resp.Err("Error while updating pack [db exec]", err, zap.String("id", id))
	}

	// Emojis are replaced wholesale, same as bots/servers above, rather than
	// incrementally patched.
	if packType == types.PackTypeEmoji {
		_, err = tx.Exec(d.Context, "DELETE FROM pack_emojis WHERE pack_url = $1", id)

		if err != nil {
			return resp.Err("Error while clearing existing pack emojis [db exec]", err, zap.String("id", id))
		}

		for i, emoji := range payload.Emojis {
			_, err = tx.Exec(
				d.Context,
				"INSERT INTO pack_emojis (id, pack_url, name, animated, position) VALUES ($1, $2, $3, $4, $5)",
				emoji.ID,
				id,
				emoji.Name,
				emoji.Animated,
				i,
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
