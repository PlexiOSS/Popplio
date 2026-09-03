// Copyright (C) 2026 NodeByte LTD

package patch_pack

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	packAssets "popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

var compiledMessages = uapi.CompileValidationErrors(PatchPack{})

type PatchPack struct {
	Name     string                   `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	Short    string                   `json:"short" validate:"required,min=10,max=100,noxss" msg:"Description must be between 10 and 100 characters"`
	Tags     []string                 `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	Bots     []string                 `json:"bots" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 bots without duplicates"`
	Servers  []string                 `json:"servers" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 servers without duplicates"`
	Emojis   []types.PackEmojiInput   `json:"emojis" validate:"omitempty,max=50,dive" msg:"There can be at most 50 emojis"`
	Stickers []types.PackStickerInput `json:"stickers" validate:"omitempty,max=50,dive" msg:"There can be at most 50 stickers"`
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
	case types.PackTypeSticker:
		if len(payload.Stickers) == 0 {
			return resp.BadRequest("A sticker pack must contain at least one sticker")
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
		oldEmojis, err := txQ.GetPackEmojis(d.Context, id)

		if err != nil {
			return resp.Err("Error while reading existing pack emojis [db fetch]", err, zap.String("id", id))
		}

		oldByID := make(map[string]db.GetPackEmojisRow, len(oldEmojis))
		for _, e := range oldEmojis {
			oldByID[e.ID] = e
		}

		newIDs := make(map[string]bool, len(payload.Emojis))
		for _, e := range payload.Emojis {
			newIDs[e.ID] = true
		}

		err = txQ.DeletePackEmojis(d.Context, id)

		if err != nil {
			return resp.Err("Error while clearing existing pack emojis [db exec]", err, zap.String("id", id))
		}

		for oldID := range oldByID {
			if newIDs[oldID] {
				continue
			}

			if err := txQ.DeleteVanityByTarget(d.Context, db.DeleteVanityByTargetParams{
				TargetID:   oldID,
				TargetType: "pack_emoji",
			}); err != nil {
				return resp.Err("Error while cleaning up a removed emoji's vanity", err, zap.String("emojiId", oldID))
			}
		}

		for i, emoji := range payload.Emojis {
			insertParams := db.InsertPackEmojiParams{
				ID:       emoji.ID,
				PackUrl:  id,
				Name:     emoji.Name,
				Animated: emoji.Animated,
				Position: int32(i),
			}

			if old, existed := oldByID[emoji.ID]; existed {
				insertParams.Downloads = pgtype.Int4{Int32: old.Downloads, Valid: true}
				insertParams.CreatedAt = old.CreatedAt
			}

			err = txQ.InsertPackEmoji(d.Context, insertParams)

			if err != nil {
				return resp.ErrBody("Failed to insert pack emoji [patch_pack]", "Failed to save one of the pack's emojis — the uploaded image may not exist yet.", err, zap.String("emojiId", emoji.ID))
			}

			if err := packAssets.EnsureDefaultVanity(d.Context, txQ, emoji.ID, "pack_emoji", emoji.Name); err != nil {
				return resp.ErrBody("Failed to set a default vanity for a pack emoji [patch_pack]", "Failed to save one of the pack's emojis.", err, zap.String("emojiId", emoji.ID))
			}
		}
	}

	if packType == types.PackTypeSticker {
		oldStickers, err := txQ.GetPackStickers(d.Context, id)

		if err != nil {
			return resp.Err("Error while reading existing pack stickers [db fetch]", err, zap.String("id", id))
		}

		oldByID := make(map[string]db.GetPackStickersRow, len(oldStickers))
		for _, s := range oldStickers {
			oldByID[s.ID] = s
		}

		newIDs := make(map[string]bool, len(payload.Stickers))
		for _, s := range payload.Stickers {
			newIDs[s.ID] = true
		}

		err = txQ.DeletePackStickers(d.Context, id)

		if err != nil {
			return resp.Err("Error while clearing existing pack stickers [db exec]", err, zap.String("id", id))
		}

		for oldID := range oldByID {
			if newIDs[oldID] {
				continue
			}

			if err := txQ.DeleteVanityByTarget(d.Context, db.DeleteVanityByTargetParams{
				TargetID:   oldID,
				TargetType: "pack_sticker",
			}); err != nil {
				return resp.Err("Error while cleaning up a removed sticker's vanity", err, zap.String("stickerId", oldID))
			}
		}

		for i, sticker := range payload.Stickers {
			insertParams := db.InsertPackStickerParams{
				ID:       sticker.ID,
				PackUrl:  id,
				Name:     sticker.Name,
				Animated: sticker.Animated,
				Position: int32(i),
			}

			if old, existed := oldByID[sticker.ID]; existed {
				insertParams.Downloads = pgtype.Int4{Int32: old.Downloads, Valid: true}
				insertParams.CreatedAt = old.CreatedAt
			}

			err = txQ.InsertPackSticker(d.Context, insertParams)

			if err != nil {
				return resp.ErrBody("Failed to insert pack sticker [patch_pack]", "Failed to save one of the pack's stickers — the uploaded image may not exist yet.", err, zap.String("stickerId", sticker.ID))
			}

			if err := packAssets.EnsureDefaultVanity(d.Context, txQ, sticker.ID, "pack_sticker", sticker.Name); err != nil {
				return resp.ErrBody("Failed to set a default vanity for a pack sticker [patch_pack]", "Failed to save one of the pack's stickers.", err, zap.String("stickerId", sticker.ID))
			}
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Failed to commit transaction [patch_pack]", err, zap.String("id", id))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
