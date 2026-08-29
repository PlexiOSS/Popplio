package patch_vanity

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

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

var compiledMessages = uapi.CompileValidationErrors(types.PatchVanity{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Entity Vanity",
		Description: "Updates an entities vanity. Returns 204 on success",
		Req:         types.PatchVanity{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	state.Logger.Info("Patch Vanity", zap.String("userID", d.Auth.ID))

	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id, target_type must be specified")
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return resp.Status(http.StatusNotImplemented, "Target type not implemented")
	}

	var payload types.PatchVanity

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	if payload.Code == "" {
		return resp.BadRequest("Vanity cannot be empty")
	}

	vanity := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, payload.Code)

	systems, err := validators.GetWordBlacklistSystems(d.Context, vanity)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return resp.BadRequest("Error while getting word blacklist systems: " + err.Error())
	}

	if slices.Contains(systems, "vanity.code") {
		return resp.BadRequest("The chosen vanity is blacklisted")
	}

	if strings.Contains(vanity, "@") {
		return resp.BadRequest("Vanity cannot contain @")
	}

	vanity = strings.TrimSuffix(vanity, "-")
	vanity = strings.ToLower(vanity)
	vanity = strings.ReplaceAll(vanity, " ", "-")

	if vanity == "" {
		return resp.BadRequest("Vanity cannot be empty")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	count, err := q.CountVanityByCode(d.Context, vanity)

	if err != nil {
		return resp.Err("Error while querying vanity", err, zap.String("userID", d.Auth.ID))
	}

	if count > 0 {
		return resp.BadRequest("Vanity is already taken")
	}

	rowCount, err := q.CountVanityByTarget(d.Context, db.CountVanityByTargetParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return resp.Err("Error while querying vanity", err, zap.String("userID", d.Auth.ID))
	}

	if rowCount == 0 {
		err = q.InsertVanity(d.Context, db.InsertVanityParams{
			TargetID:   targetId,
			TargetType: targetType,
			Code:       vanity,
		})

		if err != nil {
			return resp.Err("Error while inserting vanity", err, zap.String("userID", d.Auth.ID))
		}
	} else {
		err = q.UpdateVanityCode(d.Context, db.UpdateVanityCodeParams{
			Code:       vanity,
			TargetID:   targetId,
			TargetType: targetType,
		})

		if err != nil {
			return resp.Err("Error while updating vanity", err, zap.String("userID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
