package get_auto_vote

import (
	"errors"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
		PriceCents:       autovote.PriceCents,
		YearlyPriceCents: autovote.YearlyPriceCents,
		MaxTargets:       autovote.MaxTargets,
		MaxExtraSlots:    autovote.MaxExtraSlots,
		SlotPriceCents:   autovote.SlotPrice("month"),
		BillingInterval:  "month",
		Targets:          []types.AutoVoteTarget{},
		Notify:           "off",
	}

	if notify, err := q.GetAutoVoteNotify(d.Context, userID); err == nil {
		out.Notify = notify
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return resp.Err("Failed to load auto-vote notifications", err)
	}

	sub, err := q.GetAutoVoteSubscription(d.Context, userID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resp.Err("Failed to load auto-vote subscription", err)
	}

	if err == nil {
		out.Status = sub.Status
		out.Subscribed = autovote.Active(sub.Status, sub.CurrentPeriodEnd)
		out.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
		out.ExtraSlots = int(sub.ExtraSlots)
		out.MaxTargets = autovote.Limit(int(sub.ExtraSlots))
		out.BillingInterval = sub.BillingInterval
		out.SlotPriceCents = autovote.SlotPrice(sub.BillingInterval)

		if sub.CurrentPeriodEnd.Valid {
			end := sub.CurrentPeriodEnd.Time
			out.CurrentPeriodEnd = &end
		}
	}

	out.BoosterSlots = autovote.BoosterBonus(d.Context, userID)
	out.MaxTargets += out.BoosterSlots

	rows, err := q.ListAutoVoteTargets(d.Context, userID)

	if err != nil {
		return resp.Err("Failed to load auto-vote targets", err)
	}

	for _, row := range rows {
		out.Targets = append(out.Targets, autovote.Target(d.Context, row))
	}

	for _, window := range []struct {
		since time.Duration
		total *int
		set   func(t *types.AutoVoteTarget, n int)
	}{
		{24 * time.Hour, &out.VotesDay, func(t *types.AutoVoteTarget, n int) { t.VotesDay = n }},
		{7 * 24 * time.Hour, &out.VotesWeek, func(t *types.AutoVoteTarget, n int) { t.VotesWeek = n }},
	} {
		counts, err := q.CountUserAutoVotesSince(d.Context, db.CountUserAutoVotesSinceParams{
			Author: userID,
			Since:  pgtype.Timestamp{Time: time.Now().UTC().Add(-window.since), Valid: true},
		})

		if err != nil {
			return resp.Err("Failed to count automatic votes", err)
		}

		for _, c := range counts {
			*window.total += int(c.Votes)

			for i := range out.Targets {
				if out.Targets[i].TargetType == c.TargetType && out.Targets[i].TargetID == c.TargetID {
					window.set(&out.Targets[i], int(c.Votes))
				}
			}
		}
	}

	return uapi.HttpResponse{Json: out}
}
