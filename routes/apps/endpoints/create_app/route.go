// Package create_app implements POST /users/{user_id}/apps — "Create App For
// Position".
//
// Creates an application for a position. Returns a 204 on success.
package create_app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/PlexiOSS/Keel/ptr"
	"popplio/api/resp"
	"popplio/apps"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"

	"github.com/PlexiOSS/Keel/crypto"
)

type CreateApp struct {
	Position string            `json:"position" validate:"required"`
	Answers  map[string]string `json:"answers" validate:"required,dive,required"`
}

var compiledMessages = uapi.CompileValidationErrors(CreateApp{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create App For Position",
		Description: "Creates an application for a position. Returns a 204 on success.",
		Req:         CreateApp{},
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to create the application for.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload CreateApp

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

	position := apps.FindPosition(payload.Position)

	if position == nil {
		return resp.BadRequest("Invalid position")
	}

	if d.Auth.Banned && !position.AllowedForBanned {
		return resp.BadRequest("Banned users are not allowed to apply for this position")
	}

	if !d.Auth.Banned && position.BannedOnly {
		return resp.BadRequest("You are not banned? Why are you appealing?")
	}

	if position.Closed {
		return resp.BadRequest("This position is currently closed. Please check back later.")
	}

	q := db.New(state.Pool)

	appBanned, err := q.GetUserAppBanned(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error getting state.Pool banned state", err, zap.String("user_id", d.Auth.ID))
	}

	if appBanned {
		return resp.Forbidden("You are currently banned from making applications on the site")
	}

	userApps, err := q.CountPendingUserApps(d.Context, db.CountPendingUserAppsParams{
		UserID:   d.Auth.ID,
		Position: payload.Position,
	})

	if err != nil {
		return resp.Err("Error getting user apps", err, zap.String("user_id", d.Auth.ID), zap.String("position", payload.Position))
	}

	if userApps > 0 {
		return resp.BadRequest("You already have a pending application for this position")
	}

	if position.Cooldown > 0 {
		// Fetch the time the last app the user created was on
		lastAppTs, err := q.GetLastAppCreatedAt(d.Context, db.GetLastAppCreatedAtParams{
			UserID:   d.Auth.ID,
			Position: payload.Position,
		})
		lastApp := lastAppTs.Time

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				state.Logger.Error("Error getting last app", zap.Error(err), zap.String("user_id", d.Auth.ID), zap.String("position", payload.Position))
				return uapi.HttpResponse{
					Json: types.ApiError{
						Message: "Error getting last app: " + err.Error(),
					},
				}
			}
		} else {
			if time.Since(lastApp) < time.Duration(position.Cooldown) {
				// Get the difference between the last app and the cooldown
				waitFor := time.Since(lastApp) - time.Duration(position.Cooldown)

				return uapi.HttpResponse{
					Json: types.ApiError{
						Message: "You must wait " + waitFor.String() + " before applying for this position again",
					},
					Status: http.StatusTooManyRequests,
					Headers: map[string]string{
						"Retry-After": strconv.FormatFloat(waitFor.Seconds(), 'f', 0, 64),
					},
				}
			}
		}

	}

	var answerMap = map[string]string{}

	for _, question := range position.Questions {
		ans, ok := payload.Answers[question.ID]

		if !ok {
			return resp.BadRequest("Missing answer for question " + question.ID)
		}

		if ans == "" {
			return resp.BadRequest("Answer for question " + question.ID + " cannot be empty")
		}

		if question.Short {
			if len(ans) > 4096 {
				return resp.BadRequest("Answer for question " + question.ID + " is too long")
			}
		} else {
			if len(ans) < 50 {
				return resp.BadRequest("Answer for question " + question.ID + " is too short")
			}

			if len(ans) > 10000 {
				return resp.BadRequest("Answer for question " + question.ID + " is too long")
			}
		}

		answerMap[question.ID] = ans
	}

	var noPersistToDatabase bool
	if position.ExtraLogic != nil {
		err := position.ExtraLogic(d, *position, answerMap)

		if err != nil {
			state.Logger.Error("Error running extra logic", zap.Error(err), zap.String("user_id", d.Auth.ID), zap.String("position", payload.Position))
			return resp.BadRequest("Error: " + err.Error())
		}

		if errors.Is(err, apps.ErrNoPersist) {
			noPersistToDatabase = true
		}
	}

	var appId string
	if !noPersistToDatabase {
		appId = crypto.RandString(64)

		questionsJSON, err := json.Marshal(position.Questions)

		if err != nil {
			return resp.Err("Error marshaling questions", err, zap.String("user_id", d.Auth.ID), zap.String("position", payload.Position))
		}

		err = q.InsertApp(d.Context, db.InsertAppParams{
			AppID:     appId,
			UserID:    d.Auth.ID,
			Position:  payload.Position,
			Questions: questionsJSON,
			Answers:   answerMap,
		})

		if err != nil {
			return resp.Err("Error inserting app", err, zap.String("user_id", d.Auth.ID), zap.String("position", payload.Position))
		}
	} else {
		appId = "Not Applicable (not persisted to database)"
	}

	// Send a message to APPS channel
	var desc = "User <@" + d.Auth.ID + "> has applied for " + payload.Position + "."
	if position.PositionDescription != nil {
		desc = position.PositionDescription(d, *position)
	}

	var channel = state.Config.Channels.Apps

	if position.Channel != nil {
		channel = position.Channel()
	}

	_, err = state.Discord.Rest().CreateMessage(channel, discord.MessageCreate{
		Content: "<@&" + state.Config.Roles.Apps.String() + ">",
		Embeds: []discord.Embed{
			{
				Title: "New " + position.Name + " Application!",
				// The legacy SvelteKit panel this used to link to
				// (Sites.Panel + "/panel/apps") is superseded by
				// Omniplex's own /admin/applications.
				URL:         state.Config.Sites.Frontend + "/admin/applications",
				Description: desc,
				Color:       0x00ff00,
				Fields: []discord.EmbedField{
					{
						Name:   "App ID",
						Value:  appId,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "User ID",
						Value:  d.Auth.ID,
						Inline: ptr.TruePtr,
					},
					{
						Name:   "Position",
						Value:  payload.Position,
						Inline: ptr.TruePtr,
					},
				},
			},
		},
	})

	if err != nil {
		return resp.Err("Error sending embed to apps channel", err, zap.String("user_id", d.Auth.ID), zap.String("position", payload.Position))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
