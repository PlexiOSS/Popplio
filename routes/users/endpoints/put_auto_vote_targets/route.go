package put_auto_vote_targets

import (
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/listing"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.AutoVoteTargetsUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Auto-vote Targets",
		Description: "Replaces the listings the user auto-votes for (up to 3), each with an optional weekday and time-of-day schedule. Returns 204 on success.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.AutoVoteTargetsUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	userID := chi.URLParam(r, "id")

	var payload types.AutoVoteTargetsUpdate

	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	seen := map[string]bool{}

	for _, t := range payload.Targets {
		key := t.TargetType + ":" + t.TargetID

		if seen[key] {
			return resp.BadRequest("Each listing can only be added once")
		}

		seen[key] = true

		if _, err := time.LoadLocation(t.Timezone); err != nil {
			return resp.BadRequest("Unknown timezone: " + t.Timezone)
		}

		public, err := listing.EntityIsPublic(d.Context, t.TargetType, t.TargetID)

		if err != nil || !public {
			return resp.BadRequest("Only approved bots and servers can be auto-voted for")
		}
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Failed to start transaction", err)
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	if err := q.DeleteAutoVoteTargets(d.Context, userID); err != nil {
		return resp.Err("Failed to clear auto-vote targets", err)
	}

	for _, t := range payload.Targets {
		days := make([]int16, len(t.Days))

		for i, day := range t.Days {
			days[i] = int16(day)
		}

		if err := q.InsertAutoVoteTarget(d.Context, db.InsertAutoVoteTargetParams{
			UserID:      userID,
			TargetType:  t.TargetType,
			TargetID:    t.TargetID,
			Days:        days,
			StartMinute: int16(t.StartMinute),
			EndMinute:   int16(t.EndMinute),
			Timezone:    t.Timezone,
		}); err != nil {
			return resp.Err("Failed to save auto-vote target", err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return resp.Err("Failed to save auto-vote targets", err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
