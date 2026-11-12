// Copyright (C) 2026 NodeByte LTD

package pagestyle

import (
	"errors"
	"net/http"
	"strings"

	"popplio/api"
	"popplio/api/resp"
	"popplio/db"
	ps "popplio/pagestyle"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	updateMessages    = uapi.CompileValidationErrors(types.PageStyleUpdate{})
	submitMessages    = uapi.CompileValidationErrors(types.PageImageSubmit{})
	cardColorMessages = uapi.CompileValidationErrors(types.CardColorUpdate{})
)

func idParam(targetType string) []docs.Parameter {
	return []docs.Parameter{{
		Name:        "id",
		Description: "The " + targetType + "'s ID",
		Required:    true,
		In:          "path",
		Schema:      docs.IdSchema,
	}}
}

func errResponse(targetType string, err error) uapi.HttpResponse {
	switch {
	case errors.Is(err, ps.ErrNotFound):
		return uapi.DefaultResponse(http.StatusNotFound)
	case errors.Is(err, ps.ErrNotPremium):
		return resp.BadRequest("Page styles need premium or a Page Style pass from the Credit shop. Get one for this " + targetType + " to customise its page.")
	case errors.Is(err, ps.ErrBadGradient):
		return resp.BadRequest("That gradient isn't one of the presets or a pair of custom colours")
	case errors.Is(err, ps.ErrBadAccent):
		return resp.BadRequest("That accent colour is too dark or too light to read on both light and dark mode. Pick something in between.")
	}

	return resp.Err("Failed to save the page style", err)
}

func auth(targetType string, perm perms.Perm) ([]uapi.AuthType, map[string]any) {
	return []uapi.AuthType{
			{Type: api.TargetTypeUser},
			{Type: api.TargetTypeTeam},
		}, map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perm),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return targetType, chi.URLParam(r, "id")
				},
			},
		}
}

func Mount(r *chi.Mux, targetType, prefix string, perm perms.Perm) {
	pattern := prefix + "/{id}/page-style"

	uapi.Route{
		Pattern: pattern,
		OpId:    "get_" + targetType + "_page_style",
		Method:  uapi.GET,
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get " + targetType + " page style",
				Description: "The custom background, gradient and accent colour shown on the " + targetType + "'s page. Only approved backgrounds are returned, and nothing is returned unless the " + targetType + " has premium. Returns null when the page isn't customised.",
				Params:      idParam(targetType),
				Resp:        types.PageStyle{},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			style, err := ps.Public(d.Context, targetType, chi.URLParam(r, "id"))

			if err != nil {
				return errResponse(targetType, err)
			}

			return uapi.HttpResponse{Json: style}
		},
	}.Route(r)

	authTypes, ext := auth(targetType, perm)

	uapi.Route{
		Pattern: pattern + "/manage",
		OpId:    "get_" + targetType + "_page_style_manage",
		Method:  uapi.GET,
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get " + targetType + " page style for editing",
				Description: "The page style with everything an owner needs to edit it: a background waiting for review, why the last one was rejected, and whether the " + targetType + " has premium.",
				Params:      idParam(targetType),
				Resp:        types.ManagedPageStyle{},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			out, err := ps.Managed(d.Context, targetType, chi.URLParam(r, "id"))

			if err != nil {
				return errResponse(targetType, err)
			}

			return uapi.HttpResponse{Json: out}
		},
		Auth:    authTypes,
		ExtData: ext,
	}.Route(r)

	authTypes, ext = auth(targetType, perm)

	uapi.Route{
		Pattern: pattern,
		OpId:    "patch_" + targetType + "_page_style",
		Method:  uapi.PATCH,
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Update " + targetType + " page style",
				Description: "Sets the gradient, accent colour and how the background image is shown, or removes the image with remove_image. Needs premium. The accent colour has to be readable on both light and dark mode.",
				Params:      idParam(targetType),
				Req:         types.PageStyleUpdate{},
				Resp:        types.ApiError{},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			var payload types.PageStyleUpdate

			if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
				return hresp
			}

			if err := state.Validator.Struct(payload); err != nil {
				return uapi.ValidatorErrorResponse(updateMessages, err.(validator.ValidationErrors))
			}

			if err := ps.Update(d.Context, targetType, chi.URLParam(r, "id"), payload); err != nil {
				return errResponse(targetType, err)
			}

			return uapi.DefaultResponse(http.StatusNoContent)
		},
		Auth:    authTypes,
		ExtData: ext,
	}.Route(r)

	authTypes, ext = auth(targetType, perm)

	uapi.Route{
		Pattern: pattern + "/image",
		OpId:    "put_" + targetType + "_page_image",
		Method:  uapi.PUT,
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Submit " + targetType + " background image",
				Description: "Records a background image that was just uploaded through the website's upload gateway, which re-encodes it. It waits for staff review before it shows on the page. Needs premium.",
				Params:      idParam(targetType),
				Req:         types.PageImageSubmit{},
				Resp:        types.ApiError{},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			var payload types.PageImageSubmit

			if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
				return hresp
			}

			if err := state.Validator.Struct(payload); err != nil {
				return uapi.ValidatorErrorResponse(submitMessages, err.(validator.ValidationErrors))
			}

			if err := ps.SubmitImage(d.Context, targetType, chi.URLParam(r, "id"), payload.Version); err != nil {
				return errResponse(targetType, err)
			}

			return uapi.DefaultResponse(http.StatusNoContent)
		},
		Auth:    authTypes,
		ExtData: ext,
	}.Route(r)

	authTypes, ext = auth(targetType, perm)

	uapi.Route{
		Pattern: prefix + "/{id}/card-color",
		OpId:    "put_" + targetType + "_card_color",
		Method:  uapi.PUT,
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Set " + targetType + " card colour",
				Description: "Sets the colour that tints the " + targetType + "'s card, or resets it with an empty colour. Needs the Card Theme from the shop. Returns 204 on success.",
				Params:      idParam(targetType),
				Req:         types.CardColorUpdate{},
				Resp:        types.ApiError{},
			}
		},
		Handler: func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
			var payload types.CardColorUpdate

			if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
				return hresp
			}

			if err := state.Validator.Struct(payload); err != nil {
				return uapi.ValidatorErrorResponse(cardColorMessages, err.(validator.ValidationErrors))
			}

			id := chi.URLParam(r, "id")
			color := pgtype.Text{String: strings.ToLower(payload.Color), Valid: payload.Color != ""}
			q := db.New(state.Pool)

			var (
				n   int64
				err error
			)

			if targetType == api.TargetTypeBot {
				n, err = q.SetBotCardColor(d.Context, db.SetBotCardColorParams{BotID: id, CardColor: color})
			} else {
				n, err = q.SetServerCardColor(d.Context, db.SetServerCardColorParams{ServerID: id, CardColor: color})
			}

			if err != nil {
				return resp.Err("Failed to save the card colour", err)
			}

			if n == 0 {
				return resp.BadRequest("Card colours need the Card Theme from the shop. Get it for this " + targetType + " first.")
			}

			return resp.NoContent()
		},
		Auth:    authTypes,
		ExtData: ext,
	}.Route(r)
}
