// Package create_user_entity_vote implements PUT
// /users/{uid}/{target_type}/{target_id}/votes — "Create Entity Vote".
//
// Creates a vote for an entity. Returns 204 on success. Note that for
// compatibility, a trailing 's' is removed
package create_user_entity_vote

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"popplio/api/resp"
	"popplio/captcha"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/votes/cast"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

// maxCaptchaBodyBytes bounds how much of the request body we'll read to look
// for a captcha solution, well above any real payload's size.
const maxCaptchaBodyBytes = 4096

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Entity Vote",
		Description: "Creates a vote for an entity. Returns 204 on success. Note that for compatibility, a trailing 's' is removed. If the target is a bot or server that has not opted out of captchas, the request body must contain a solved captcha.Solution obtained from GET /votes/captcha/challenge",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "upvote",
				Description: "Whether or not to upvote the entity. Must be either true or false",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	// Check if upvote query parameter is valid
	upvoteStr := r.URL.Query().Get("upvote")

	if upvoteStr != "true" && upvoteStr != "false" {
		return resp.BadRequest("upvote must be either `true` or `false`")
	}

	upvote := upvoteStr == "true"

	// Check if user is allowed to even make a vote right now.
	voteBanned, err := db.New(state.Pool).GetUserVoteBanned(d.Context, d.Auth.ID)

	if err != nil {
		state.Logger.Error("Failed to check if user is vote banned", zap.Error(err), zap.String("userId", d.Auth.ID))
		return resp.BadRequest("Error checking if user is vote banned.")
	}

	if voteBanned {
		return resp.Forbidden("You are banned from voting right now! Contact support if you think this is a mistake")
	}

	// Bots/servers require a solved captcha unless the owner has opted out
	// (captcha_opt_out). The solution, if any, travels as a JSON body; it's
	// optional at the transport level since most target types don't need one.
	requiresCaptcha, err := captcha.RequiresCaptcha(d.Context, targetType, targetId)

	if err != nil {
		return resp.ErrBody("Failed to check captcha requirement [create_user_entity_vote]", "Failed to check whether this vote requires a captcha.", err, zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	if requiresCaptcha {
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxCaptchaBodyBytes+1))
		r.Body.Close()

		if err != nil {
			return resp.ErrBody("Failed to read captcha solution body [create_user_entity_vote]", "Failed to read request body.", err)
		}

		if len(bodyBytes) == 0 || len(bodyBytes) > maxCaptchaBodyBytes {
			return resp.BadRequest("This vote requires solving a captcha first. Fetch a challenge from GET /votes/captcha/challenge, solve it, and submit the solution as the request body")
		}

		var solution captcha.Solution

		if err := json.Unmarshal(bodyBytes, &solution); err != nil {
			return resp.BadRequest("Invalid captcha solution payload")
		}

		if err := captcha.Verify(d.Context, solution); err != nil {
			return resp.BadRequest("Captcha verification failed: " + err.Error())
		}
	}

	res, err := cast.Cast(d.Context, d.Auth.ID, targetType, targetId, upvote, false)

	if err != nil {
		var castErr *cast.Error

		if errors.As(err, &castErr) {
			return resp.BadRequest(castErr.Message)
		}

		return resp.Err("Failed to cast vote", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	cast.Announce(d.Auth.ID, targetType, targetId, res, false)

	return uapi.DefaultResponse(http.StatusNoContent)
}
