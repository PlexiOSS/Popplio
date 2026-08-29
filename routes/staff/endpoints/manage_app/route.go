// Package manage_app implements PATCH /staff/apps/{app_id} — "Staff: Manage
// Application".
//
// Approves or denies an application. Returns a 204 on success.
package manage_app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/PlexiOSS/Keel/ptr"
	"popplio/api/resp"
	"popplio/apps"
	"popplio/db"
	"popplio/notifications"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type ManageApp struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason" validate:"required,min=5,max=1000" msg:"Reason must be between 5 and 1000 characters long"`
}

var (
	compiledMessages = uapi.CompileValidationErrors(ManageApp{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Manage Application",
		Description: "Approves or denies an application. Returns a 204 on success.",
		Req:         ManageApp{},
		Params: []docs.Parameter{
			{
				Name:        "app_id",
				Description: "The App ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	staffPerms, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	// Check if the user has the permission to view apps
	if !staffPerms.Has(perms.StaffManageApps) {
		return resp.Forbidden("You do not have permission to manage apps.")
	}

	var payload ManageApp

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	// Fetch app info such as the position from database
	appId := chi.URLParam(r, "app_id")

	q := db.New(state.Pool)

	row, err := q.GetAppByID(d.Context, appId)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Failed to fetch application info", err, zap.String("appId", appId))
	}

	var questions []types.Question
	if err := json.Unmarshal(row.Questions, &questions); err != nil {
		return resp.Err("Failed to parse application questions [json]", err, zap.String("appId", appId))
	}

	var reviewFeedback *string
	if row.ReviewFeedback.Valid {
		reviewFeedback = &row.ReviewFeedback.String
	}

	app := types.AppResponse{
		AppID:          row.AppID,
		UserID:         row.UserID,
		Questions:      questions,
		Answers:        row.Answers,
		State:          row.State,
		CreatedAt:      row.CreatedAt.Time,
		Position:       row.Position,
		ReviewFeedback: reviewFeedback,
	}

	if app.State != "pending" {
		return resp.BadRequest("This app is not pending approval")
	}

	position := apps.FindPosition(app.Position)

	if position == nil {
		// Delete the app from the database
		err = q.DeleteApp(d.Context, appId)

		if err != nil {
			return resp.Err("Failed to delete app", err, zap.String("appId", appId))
		}

		return resp.BadRequest("This position doesn't exist and so the app has been deleted.")
	}

	var embeds []discord.Embed

	if payload.Approved {
		if position.ReviewLogic != nil {
			err := position.ReviewLogic(d, app, payload.Reason, true)

			if err != nil {
				state.Logger.Error("Error running review logic", zap.Error(err), zap.String("appId", appId))
				return resp.BadRequest("Error: " + err.Error())
			}
		}

		err = q.ApproveApp(d.Context, db.ApproveAppParams{
			AppID:          appId,
			ReviewFeedback: pgtype.Text{String: payload.Reason, Valid: true},
		})

		if err != nil {
			return resp.Err("Failed to update app", err, zap.String("appId", appId))
		}

		embeds = []discord.Embed{
			{
				Title: "Application Approved",
				// The legacy SvelteKit panel (Sites.Panel + "/panel/apps") is
				// superseded by Omniplex's own /admin/applications.
				URL:         state.Config.Sites.Frontend + "/admin/applications",
				Description: fmt.Sprintf("<@%s> has approved an application by <@%s> for the position of %s", d.Auth.ID, app.UserID, app.Position),
				Color:       0x00ff00,
				Fields: []discord.EmbedField{
					{
						Name:   "App ID",
						Value:  appId,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "User ID",
						Value:  app.UserID,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Approved By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Position",
						Value:  app.Position,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Feedback",
						Value:  payload.Reason,
						Inline: ptr.TruePtr,
					},
				},
			},
		}
	} else {
		if position.ReviewLogic != nil {
			err := position.ReviewLogic(d, app, payload.Reason, false)

			if err != nil {
				state.Logger.Error("Error running review logic", zap.Error(err), zap.String("appId", appId))
				return resp.BadRequest("Error: " + err.Error())
			}
		}

		err = q.DenyApp(d.Context, db.DenyAppParams{
			AppID:          appId,
			ReviewFeedback: pgtype.Text{String: payload.Reason, Valid: true},
		})

		if err != nil {
			return resp.Err("Failed to update app", err, zap.String("appId", appId))
		}

		embeds = []discord.Embed{
			{
				Title: "Application Denied",
				// The legacy SvelteKit panel (Sites.Panel + "/panel/apps") is
				// superseded by Omniplex's own /admin/applications.
				URL:         state.Config.Sites.Frontend + "/admin/applications",
				Description: fmt.Sprintf("<@%s> has denied an application by <@%s> for the position of %s", d.Auth.ID, app.UserID, app.Position),
				Color:       0xff0000,
				Fields: []discord.EmbedField{
					{
						Name:   "App ID",
						Value:  appId,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "User ID",
						Value:  app.UserID,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Denied By",
						Value:  fmt.Sprintf("<@%s>", d.Auth.ID),
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Position",
						Value:  app.Position,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Reason",
						Value:  payload.Reason,
						Inline: ptr.TruePtr,
					},
				},
			},
		}
	}

	// Send message to apps channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.Apps, discord.MessageCreate{
		Embeds: embeds,
	})

	if err != nil {
		return resp.Err("Failed to send message to apps channel", err, zap.String("appId", appId))
	}

	alertTitle := "Application Denied"
	alertMessage := "Your application for " + app.Position + " was denied. " + payload.Reason
	alertType := types.AlertTypeError

	if payload.Approved {
		alertTitle = "Application Approved"
		alertMessage = "Your application for " + app.Position + " was approved!"
		alertType = types.AlertTypeSuccess
	}

	if err := notifications.PushNotification(app.UserID, types.Alert{
		Type:     alertType,
		Title:    alertTitle,
		Message:  alertMessage,
		URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/apps/" + appId, Valid: true},
		Category: types.AlertCategoryStaffApplications,
	}); err != nil {
		state.Logger.Warn("Failed to notify applicant of application verdict", zap.Error(err), zap.String("appId", appId), zap.String("userId", app.UserID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
