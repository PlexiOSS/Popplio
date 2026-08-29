// Package patch_user_alerts implements PATCH /users/{id}/alerts — "Patch
// User Alerts".
//
// Updates a set of user alerts with a given 'patch' to apply to the alert.
// Returns 204 on success
package patch_user_alerts

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var compiledMessages = uapi.CompileValidationErrors(types.AlertPatch{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Patch User Alerts",
		Description: "Updates a set of user alerts with a given 'patch' to apply to the alert. Returns 204 on success",
		Req:         types.AlertPatch{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.AlertPatch

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

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	for _, patch := range payload.Patches {
		var itagUUID pgtype.UUID
		if err := itagUUID.Scan(patch.ITag); err != nil {
			return resp.Err("Invalid itag in patch", err, zap.Any("patch", patch), zap.String("userID", d.Auth.ID))
		}

		switch patch.Patch {
		case "ack":
			err = q.AckAlert(d.Context, db.AckAlertParams{UserID: d.Auth.ID, Itag: itagUUID})
		case "unack":
			err = q.UnackAlert(d.Context, db.UnackAlertParams{UserID: d.Auth.ID, Itag: itagUUID})
		case "delete":
			err = q.DeleteUserAlert(d.Context, db.DeleteUserAlertParams{UserID: d.Auth.ID, Itag: itagUUID})
		}

		if err != nil {
			return resp.Err("Error while patching user alerts", err, zap.Any("patch", patch), zap.String("userID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
