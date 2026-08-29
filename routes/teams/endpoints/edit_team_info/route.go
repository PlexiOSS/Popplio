package edit_team_info

import (
	"net/http"

	"encoding/json"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	cevents "popplio/webhooks/core/events"
	"popplio/webhooks/events"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateEditTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Team Info",
		Description: "Edits a team. Returns a 204 on success.",
		Params: []docs.Parameter{
			{
				Name:        "tid",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.CreateEditTeam{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload types.CreateEditTeam

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error beginning transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	oldInfo, err := q.GetTeamInfoForEdit(d.Context, teamId)

	if err != nil {
		return resp.Err("Error getting team info [db queryrow]", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	oldName, oldShort, oldTags, oldNsfw := oldInfo.Name, oldInfo.Short, oldInfo.Tags, oldInfo.Nsfw

	var oldExtraLinks []types.Link
	if err := json.Unmarshal(oldInfo.ExtraLinks, &oldExtraLinks); err != nil {
		return resp.Err("Error parsing team extra_links [json]", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	err = q.TouchTeamUpdatedAt(d.Context, teamId)

	if err != nil {
		return resp.Err("Error updating team updated_at", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	err = q.UpdateTeamName(d.Context, db.UpdateTeamNameParams{Name: payload.Name, ID: teamId})

	if err != nil {
		return resp.Err("Error updating team info", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	if payload.Short != nil {
		err = q.UpdateTeamShort(d.Context, db.UpdateTeamShortParams{
			Short: pgtype.Text{String: *payload.Short, Valid: true},
			ID:    teamId,
		})

		if err != nil {
			return resp.Err("Error updating team info", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		}
	}

	if payload.Tags != nil {
		err = q.UpdateTeamTags(d.Context, db.UpdateTeamTagsParams{Tags: *payload.Tags, ID: teamId})

		if err != nil {
			return resp.Err("Error updating team info", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		}
	}

	if payload.ExtraLinks != nil {
		err = validators.ValidateExtraLinks(*payload.ExtraLinks)

		if err != nil {
			return resp.BadRequest(err.Error())
		}

		extraLinksJSON, err := json.Marshal(*payload.ExtraLinks)

		if err != nil {
			return resp.Err("Error marshaling extra links", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		}

		err = q.UpdateTeamExtraLinks(d.Context, db.UpdateTeamExtraLinksParams{ExtraLinks: extraLinksJSON, ID: teamId})

		if err != nil {
			return resp.Err("Error updating team info", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		}
	}

	var isTeamNsfw = false
	if payload.NSFW != nil {
		isTeamNsfw = *payload.NSFW
	}

	if payload.Tags != nil {
		tagList := *payload.Tags

		for _, tag := range tagList {
			if cases.Lower(language.English).String(tag) == "nsfw" {
				isTeamNsfw = true
			}
		}
	}

	if isTeamNsfw != oldNsfw {
		err = q.UpdateTeamNsfw(d.Context, db.UpdateTeamNsfwParams{Nsfw: isTeamNsfw, ID: teamId})

		if err != nil {
			return resp.Err("Error updating team info", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	err = drivers.Send(drivers.With{
		Data: events.WebhookTeamEditData{
			Name: cevents.Changeset[string]{
				Old: oldName,
				New: payload.Name,
			},
			Short: func() cevents.Changeset[string] {
				if payload.Short == nil {
					return cevents.Changeset[string]{
						Old: oldShort.String,
						New: "",
					}
				}

				return cevents.Changeset[string]{
					Old: oldShort.String,
					New: *payload.Short,
				}
			}(),
			Tags: func() cevents.Changeset[[]string] {
				if payload.Tags == nil {
					return cevents.Changeset[[]string]{}
				}

				return cevents.Changeset[[]string]{
					Old: oldTags,
					New: *payload.Tags,
				}
			}(),
			ExtraLinks: func() cevents.Changeset[[]types.Link] {
				if payload.ExtraLinks == nil {
					return cevents.Changeset[[]types.Link]{
						Old: oldExtraLinks,
						New: []types.Link{},
					}
				}

				return cevents.Changeset[[]types.Link]{
					Old: oldExtraLinks,
					New: *payload.ExtraLinks,
				}
			}(),
			NSFW: cevents.Changeset[bool]{
				Old: oldNsfw,
				New: isTeamNsfw,
			},
		},
		UserID:     d.Auth.ID,
		TargetType: "team",
		TargetID:   teamId,
	})

	if err != nil {
		state.Logger.Error("Error sending team edit webhook", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
