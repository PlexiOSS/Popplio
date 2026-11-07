package get_auto_vote_overview

import (
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/autovote"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Get Auto-vote Overview",
		Description: "Lists every Auto-vote subscription and saved listing with its status. Needs `view_payments`.",
		Resp:        types.AutoVoteStaffOverview{},
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
		return resp.Err("Error while loading staff permissions", err)
	}

	if !staffPerms.Has(perms.StaffViewPayments) {
		return resp.Forbidden("You do not have permission to view payments [view_payments]")
	}

	q := db.New(state.Pool)

	subs, err := q.ListAllAutoVoteSubscriptions(d.Context)

	if err != nil {
		return resp.Err("Failed to load subscriptions", err)
	}

	targets, err := q.ListAllAutoVoteTargets(d.Context)

	if err != nil {
		return resp.Err("Failed to load targets", err)
	}

	counts, err := q.CountAutoVotesSince(d.Context, pgtype.Timestamp{Time: time.Now().UTC().Add(-30 * 24 * time.Hour), Valid: true})

	if err != nil {
		return resp.Err("Failed to count automatic votes", err)
	}

	votes := make(map[string]int, len(counts))

	for _, c := range counts {
		votes[c.Author] = int(c.Votes)
	}

	entries := []types.AutoVoteStaffEntry{}
	index := map[string]int{}

	for _, s := range subs {
		entry := types.AutoVoteStaffEntry{
			UserID:               s.UserID,
			Status:               s.Status,
			Active:               autovote.Active(s.Status, s.CurrentPeriodEnd),
			CancelAtPeriodEnd:    s.CancelAtPeriodEnd,
			StripeSubscriptionID: s.StripeSubscriptionID,
			StripeCustomerID:     s.StripeCustomerID,
			AutoVotes30d:         votes[s.UserID],
			ExtraSlots:           int(s.ExtraSlots),
			BillingInterval:      s.BillingInterval,
			Targets:              []types.AutoVoteTarget{},
		}

		if s.CurrentPeriodEnd.Valid {
			end := s.CurrentPeriodEnd.Time
			entry.CurrentPeriodEnd = &end
		}

		if s.CreatedAt.Valid {
			at := s.CreatedAt.Time
			entry.SubscribedAt = &at
		}

		index[s.UserID] = len(entries)
		entries = append(entries, entry)
	}

	for _, t := range targets {
		i, ok := index[t.UserID]

		if !ok {
			i = len(entries)
			index[t.UserID] = i
			entries = append(entries, types.AutoVoteStaffEntry{UserID: t.UserID, Targets: []types.AutoVoteTarget{}})
		}

		entries[i].Targets = append(entries[i].Targets, autovote.Target(d.Context, t))
	}

	return uapi.HttpResponse{Json: types.AutoVoteStaffOverview{Entries: entries}}
}
