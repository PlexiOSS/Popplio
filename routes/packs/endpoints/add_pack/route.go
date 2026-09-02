// Copyright (C) 2026 NodeByte LTD

package add_pack

import (
	"net/http"
	"slices"
	"strings"
	"unicode"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(CreatePack{})

type CreatePack struct {
	Name     string                   `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	URL      string                   `json:"url" validate:"required,min=3,max=20,nospaces,notblank,alpha" msg:"URL must be between 3 and 20 characters without spaces and must be alphabetic"`
	Short    string                   `json:"short" validate:"required,min=10,max=100,noxss" msg:"Description must be between 10 and 100 characters"`
	Tags     []string                 `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	PackType string                   `json:"pack_type" validate:"required,oneof=bot server emoji sticker" msg:"pack_type must be one of bot, server, emoji, or sticker"`
	Bots     []string                 `json:"bots" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 bots without duplicates"`
	Servers  []string                 `json:"servers" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 servers without duplicates"`
	Emojis   []types.PackEmojiInput   `json:"emojis" validate:"omitempty,max=50,dive" msg:"There can be at most 50 emojis"`
	Stickers []types.PackStickerInput `json:"stickers" validate:"omitempty,max=50,dive" msg:"There can be at most 50 stickers"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Pack",
		Description: "Creates a pack. Returns 204 on success",
		Req:         CreatePack{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload CreatePack

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	switch payload.PackType {
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

	if payload.Bots == nil {
		payload.Bots = []string{}
	}
	if payload.Servers == nil {
		payload.Servers = []string{}
	}

	payload.URL = strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, payload.URL)

	systems, err := validators.GetWordBlacklistSystems(d.Context, payload.URL)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return resp.BadRequest("Error while getting word blacklist systems: " + err.Error())
	}

	if slices.Contains(systems, "pack.url") {
		return resp.BadRequest("The chosen pack url is blacklisted")
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

	q := db.New(state.Pool)

	for _, server := range payload.Servers {
		count, err := q.CountServerByID(d.Context, server)

		if err != nil {
			return resp.ErrBody("Error checking if server exists:", "Error checking if server exists: "+err.Error(), err)
		}

		if count == 0 {
			return resp.BadRequest("One of the servers you wish to add does not exist [" + server + "]")
		}
	}

	count, err := q.CountPackByURL(d.Context, payload.URL)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	if count > 0 {
		return resp.BadRequest("A pack with that URL already exists")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.ErrBody("Failed to create transaction [add_pack]", "Failed to create transaction.", err)
	}

	defer tx.Rollback(d.Context)

	txQ := db.New(tx)

	err = txQ.InsertPack(d.Context, db.InsertPackParams{
		Name:     payload.Name,
		Url:      payload.URL,
		Short:    payload.Short,
		Tags:     payload.Tags,
		Bots:     payload.Bots,
		Servers:  payload.Servers,
		Owner:    d.Auth.ID,
		PackType: payload.PackType,
	})

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	for i, emoji := range payload.Emojis {
		err = txQ.InsertPackEmoji(d.Context, db.InsertPackEmojiParams{
			ID:       emoji.ID,
			PackUrl:  payload.URL,
			Name:     emoji.Name,
			Animated: emoji.Animated,
			Position: int32(i),
		})

		if err != nil {
			return resp.ErrBody("Failed to insert pack emoji [add_pack]", "Failed to save one of the pack's emojis — the uploaded image may not exist yet.", err, zap.String("emojiId", emoji.ID))
		}
	}

	for i, sticker := range payload.Stickers {
		err = txQ.InsertPackSticker(d.Context, db.InsertPackStickerParams{
			ID:       sticker.ID,
			PackUrl:  payload.URL,
			Name:     sticker.Name,
			Animated: sticker.Animated,
			Position: int32(i),
		})

		if err != nil {
			return resp.ErrBody("Failed to insert pack sticker [add_pack]", "Failed to save one of the pack's stickers — the uploaded image may not exist yet.", err, zap.String("stickerId", sticker.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Failed to commit transaction [add_pack]", err, zap.String("userId", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
