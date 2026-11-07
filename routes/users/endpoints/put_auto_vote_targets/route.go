package put_auto_vote_targets

import (
	"net/http"
	"strconv"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
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
		Description: "Replaces the listings the user auto-votes for (3, plus any extra slots they've bought), each with an optional weekday and time-of-day schedule and a pause switch, and optionally sets how the user hears about automatic votes. Returns 204 on success.",
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

	extra, err := db.New(state.Pool).GetAutoVoteExtraSlots(d.Context, userID)

	if err != nil {
		return resp.Err("Failed to check auto-vote slots", err)
	}

	if limit := autovote.Limit(int(extra)) + autovote.BoosterBonus(d.Context, userID); len(payload.Targets) > limit {
		return resp.BadRequest("You can auto-vote for up to " + strconv.Itoa(limit) + " listings. Add more slots to auto-vote for more.")
	}

	seen := map[string]bool{}
	keep := make([]string, 0, len(payload.Targets))

	for _, t := range payload.Targets {
		key := t.TargetType + ":" + t.TargetID

		if seen[key] {
			return resp.BadRequest("Each listing can only be added once")
		}

		seen[key] = true
		keep = append(keep, key)

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

	if err := q.DeleteAutoVoteTargetsExcept(d.Context, db.DeleteAutoVoteTargetsExceptParams{UserID: userID, Keep: keep}); err != nil {
		return resp.Err("Failed to clear auto-vote targets", err)
	}

	for _, t := range payload.Targets {
		days := make([]int16, len(t.Days))

		for i, day := range t.Days {
			days[i] = int16(day)
		}

		if err := q.UpsertAutoVoteTarget(d.Context, db.UpsertAutoVoteTargetParams{
			UserID:      userID,
			TargetType:  t.TargetType,
			TargetID:    t.TargetID,
			Days:        days,
			StartMinute: int16(t.StartMinute),
			EndMinute:   int16(t.EndMinute),
			Timezone:    t.Timezone,
			Paused:      t.Paused,
		}); err != nil {
			return resp.Err("Failed to save auto-vote target", err)
		}
	}

	if payload.Notify != nil {
		if err := q.SetAutoVoteNotify(d.Context, db.SetAutoVoteNotifyParams{UserID: userID, Notify: *payload.Notify}); err != nil {
			return resp.Err("Failed to save auto-vote notifications", err)
		}
	}

	if err := tx.Commit(d.Context); err != nil {
		return resp.Err("Failed to save auto-vote targets", err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
