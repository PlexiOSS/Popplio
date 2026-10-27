package get_auto_vote

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Auto-vote",
		Description: "Returns the user's auto-vote subscription and the listings they auto-vote for. Only the user themselves can check.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AutoVoteStatus{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	userID := chi.URLParam(r, "id")
	q := db.New(state.Pool)

	out := types.AutoVoteStatus{
		PriceCents: autovote.PriceCents,
		MaxTargets: autovote.MaxTargets,
		Targets:    []types.AutoVoteTarget{},
	}

	sub, err := q.GetAutoVoteSubscription(d.Context, userID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resp.Err("Failed to load auto-vote subscription", err)
	}

	if err == nil {
		out.Status = sub.Status
		out.Subscribed = autovote.Active(sub.Status, sub.CurrentPeriodEnd)
		out.CancelAtPeriodEnd = sub.CancelAtPeriodEnd

		if sub.CurrentPeriodEnd.Valid {
			end := sub.CurrentPeriodEnd.Time
			out.CurrentPeriodEnd = &end
		}
	}

	rows, err := q.ListAutoVoteTargets(d.Context, userID)

	if err != nil {
		return resp.Err("Failed to load auto-vote targets", err)
	}

	for _, row := range rows {
		target := types.AutoVoteTarget{
			TargetType:  row.TargetType,
			TargetID:    row.TargetID,
			Name:        row.TargetID,
			Days:        make([]int, len(row.Days)),
			StartMinute: int(row.StartMinute),
			EndMinute:   int(row.EndMinute),
			Timezone:    row.Timezone,
		}

		for i, day := range row.Days {
			target.Days[i] = int(day)
		}

		if info, err := votes.GetEntityInfo(d.Context, state.Pool, row.TargetID, row.TargetType); err == nil {
			target.Name = info.Name
			target.Avatar = info.Avatar
		}

		if row.LastVotedAt.Valid {
			at := row.LastVotedAt.Time
			target.LastVotedAt = &at
		}

		if row.LastSkipReason.Valid {
			reason := row.LastSkipReason.String
			target.LastSkipReason = &reason
		}

		out.Targets = append(out.Targets, target)
	}

	return uapi.HttpResponse{Json: out}
}
