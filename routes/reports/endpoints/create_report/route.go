// Package create_report implements PUT
// /users/{uid}/{target_type}/{target_id}/reports — "Create Report".
//
// Files a report against any votable entity. Returns 204 on success.
// Reporter identity is never exposed outside the staff panel — see
// arcadia/panel/ops_reports.go.
package create_report

import (
	"net/http"
	"popplio/api/resp"
	"popplio/reports"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// dailyReportLimit caps how many reports a single user may file across all
// targets in a 24h window. The partial unique index on (target_type,
// target_id, reporter_id) already stops someone spamming *one* target;
// this stops spamming many different targets instead.
const dailyReportLimit = 10

var compiledMessages = uapi.CompileValidationErrors(types.CreateReport{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Report",
		Description: "Files a report against any votable entity. Returns 204 on success",
		Req:         types.CreateReport{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The reporting user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity being reported",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The ID of the entity being reported",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	var payload types.CreateReport

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	targetInfo, err := reports.GetTargetInfo(d.Context, state.Pool, targetType, targetId)

	if err != nil {
		return resp.BadRequest("Could not file report: " + err.Error())
	}

	var dailyCount int64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND created_at > NOW() - INTERVAL '24 hours'", d.Auth.ID).Scan(&dailyCount)

	if err != nil {
		return resp.ErrBody("Failed to check daily report count [create_report]", "Failed to file report.", err, zap.String("userId", d.Auth.ID))
	}

	if dailyCount >= dailyReportLimit {
		return resp.BadRequest("You've filed too many reports today. Please try again tomorrow")
	}

	_, err = state.Pool.Exec(
		d.Context,
		"INSERT INTO reports (target_type, target_id, reporter_id, reason, description) VALUES ($1, $2, $3, $4, $5)",
		targetType,
		targetId,
		d.Auth.ID,
		payload.Reason,
		payload.Description,
	)

	if err != nil {
		// The partial unique index (one open report per reporter per target)
		// is the only constraint that can realistically fail here.
		return resp.BadRequest("You already have an open report on this")
	}

	// Staff-only channel — reporter identity is never exposed outside the
	// staff panel (see this file's own doc comment), and this message
	// doesn't include it either, only the target and reason. Best-effort:
	// the report is already filed, so a failed Discord post shouldn't turn
	// into an error response.
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.StaffLogs, discord.MessageCreate{
		Content: "New report filed against **" + targetInfo.Name + "** (" + targetType + ", " + payload.Reason + "): " + targetInfo.URL,
	})

	if err != nil {
		state.Logger.Warn("Failed to post new-report staff log", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
