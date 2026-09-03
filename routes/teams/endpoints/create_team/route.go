// Copyright (C) 2026 NodeByte LTD

package create_team

import (
	"encoding/json"
	"net/http"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/PlexiOSS/Keel/crypto"
	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-playground/validator/v10"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateEditTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Team",
		Description: "Creates a team. Returns a 201 with the team ID on success.",
		Params:      []docs.Parameter{},
		Req:         types.CreateEditTeam{},
		Resp:        types.CreateTeamResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
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

	var el = []types.Link{}

	if payload.ExtraLinks != nil {
		err = validators.ValidateExtraLinks(*payload.ExtraLinks)

		if err != nil {
			return resp.BadRequest(err.Error())
		}

		el = *payload.ExtraLinks
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

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error starting transaction", err, zap.String("user_id", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	vanity := strings.ToLower(payload.Name)

	var repl = [][2]string{
		{" ", "-"},
		{"_", "-"},
		{".", ""},
	}

	for _, r := range repl {
		vanity = strings.ReplaceAll(vanity, r[0], r[1])
	}

	count, err := q.CountVanityByCode(d.Context, vanity)

	if err != nil {
		return resp.Err("Error while checking vanity", err, zap.String("userID", d.Auth.ID), zap.String("vanity", vanity))
	}

	for count > 0 {
		newVanity := vanity + "-" + crypto.RandString(8)

		nc, err := q.CountVanityByCode(d.Context, newVanity)

		if err != nil {
			return resp.Err("Error while checking vanity", err, zap.String("userID", d.Auth.ID), zap.String("vanity", vanity))
		}

		if nc == 0 {
			vanity = newVanity
			break
		}
	}

	var teamId = uuid.New().String()

	if teamId == "" {
		return resp.Err("Error generating team ID", err, zap.String("user_id", d.Auth.ID))
	}

	itag, err := q.InsertVanityReturningItag(d.Context, db.InsertVanityReturningItagParams{
		Code:       vanity,
		TargetID:   teamId,
		TargetType: "team",
	})

	if err != nil {
		return resp.Err("Error while inserting vanity", err, zap.String("userID", d.Auth.ID), zap.String("teamId", teamId), zap.String("vanity", vanity))
	}

	extraLinksJSON, err := json.Marshal(el)

	if err != nil {
		return resp.Err("Error marshaling extra links", err, zap.String("user_id", d.Auth.ID))
	}

	shortText := pgtype.Text{}
	if payload.Short != nil {
		shortText = pgtype.Text{String: *payload.Short, Valid: true}
	}

	var tags []string
	if payload.Tags != nil {
		tags = *payload.Tags
	}

	err = q.InsertTeam(d.Context, db.InsertTeamParams{
		ID:         teamId,
		Name:       payload.Name,
		Short:      shortText,
		Tags:       tags,
		ExtraLinks: extraLinksJSON,
		Nsfw:       isTeamNsfw,
		VanityRef:  itag,
	})

	if err != nil {
		return resp.Err("Error creating team", err, zap.String("user_id", d.Auth.ID))
	}

	var teamIdUUID pgtype.UUID
	if err := teamIdUUID.Scan(teamId); err != nil {
		return resp.Err("Invalid team ID", err, zap.String("user_id", d.Auth.ID), zap.String("teamId", teamId))
	}

	err = q.InsertTeamMemberOwner(d.Context, db.InsertTeamMemberOwnerParams{
		TeamID: teamIdUUID,
		UserID: d.Auth.ID,
		Flags:  []string{perms.EntityOwner.String()},
	})

	if err != nil {
		return resp.Err("Error adding user to team", err, zap.String("user_id", d.Auth.ID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("user_id", d.Auth.ID))
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: types.CreateTeamResponse{
			TeamID: teamId,
		},
	}
}
