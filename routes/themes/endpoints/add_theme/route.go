// Copyright (C) 2026 NodeByte LTD

package add_theme

import (
	"net/http"
	"slices"

	"popplio/api/resp"
	"popplio/db"
	"popplio/moderation"
	"popplio/state"
	"popplio/validators"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var compiledMessages = uapi.CompileValidationErrors(CreateTheme{})

type CreateTheme struct {
	Name           string   `json:"name" validate:"required,min=3,max=40,notblank,nonvulgar,noxss" msg:"Name must be between 3 and 40 characters"`
	PrimaryColor   string   `json:"primary_color" validate:"required,hex6" msg:"Primary color must be a 6-digit hex code, e.g. #5865F2"`
	SecondaryColor string   `json:"secondary_color" validate:"required,hex6" msg:"Secondary color must be a 6-digit hex code, e.g. #5865F2"`
	Tags           []string `json:"tags" validate:"required,unique,min=1,max=3,dive,oneof=Green Blue Purple Pink Red Orange Dark Light Gradient Aesthetic Minimal Vibrant" msg:"There must be between 1 and 3 categories without duplicates" amsg:"Each category must be one of the listed options"`
}

type CreateThemeResponse struct {
	ID string `json:"id" description:"The created theme's ID"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Theme",
		Description: "Submits a Discord profile theme -- a name plus two hex colors and up to 3 categories",
		Req:         CreateTheme{},
		Resp:        CreateThemeResponse{},
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
	var payload CreateTheme

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errs := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errs)
	}

	systems, err := validators.GetWordBlacklistSystems(d.Context, payload.Name)

	if err != nil {
		return resp.Err("Error while getting word blacklist systems", err)
	}

	if slices.Contains(systems, "theme.name") {
		return resp.BadRequest("The chosen name is blacklisted")
	}

	id := uuid.New().String()

	q := db.New(state.Pool)

	if err := q.InsertTheme(d.Context, db.InsertThemeParams{
		ID:             id,
		Name:           payload.Name,
		PrimaryColor:   payload.PrimaryColor,
		SecondaryColor: payload.SecondaryColor,
		Tags:           payload.Tags,
		Owner:          d.Auth.ID,
	}); err != nil {
		return resp.Err("Error inserting theme", err)
	}

	if result, err := moderation.CheckText(d.Context, payload.Name); err != nil {
		state.Logger.Error("Failed to run moderation check on new theme", zap.Error(err), zap.String("id", id))
	} else if result.Flagged {
		if err := moderation.FileAutoReport(d.Context, "theme", id, result.Categories); err != nil {
			state.Logger.Error("Failed to auto-file report for flagged theme", zap.Error(err), zap.String("id", id))
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: CreateThemeResponse{
			ID: id,
		},
	}
}
