package audit

import (
	"context"
	"strings"
	"testing"

	"popplio/db"
	"popplio/testdb"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestTarget(t *testing.T) {
	cases := []struct {
		pattern  string
		params   map[string]string
		wantType string
		wantID   string
	}{
		{"/{target_type}/{target_id}/webhooks", map[string]string{"target_type": "bot", "target_id": "1"}, "bot", "1"},
		{"/users/{id}/stripe", map[string]string{"id": "2"}, "user", "2"},
		{"/bots/{id}/settings", map[string]string{"id": "3"}, "bot", "3"},
		{"/servers/{server_id}", map[string]string{"server_id": "4"}, "server", "4"},
		{"/payments/stripe/webhook", map[string]string{}, "", ""},
		{"/unknown/{id}", map[string]string{"id": "5"}, "", ""},
	}

	for _, c := range cases {
		gotType, gotID := target(c.pattern, c.params)

		if gotType != c.wantType || gotID != c.wantID {
			t.Errorf("%s: got %q %q, want %q %q", c.pattern, gotType, gotID, c.wantType, c.wantID)
		}
	}
}

func TestWriteAndList(t *testing.T) {
	testdb.Pool(t)

	ctx := context.Background()
	actor := testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM audit_events WHERE actor_id = $1", actor)

	events := []Event{
		{Source: SourceAPI, ActorType: ActorUser, ActorID: actor, Action: "patch_bot_settings", TargetType: "bot", TargetID: "1", Status: 200},
		{Source: SourceAPI, ActorType: ActorUser, ActorID: actor, Action: "patch_bot_settings", TargetType: "bot", TargetID: "1", Status: 403},
		{Source: SourceStripe, ActorType: ActorUser, ActorID: actor, Action: "payment.refunded", TargetType: "bot", TargetID: "2", Data: map[string]any{"credits_returned": 10}},
		{Source: SourceSystem, ActorType: ActorUser, ActorID: actor, Action: "autovote.voted", Data: map[string]any{"blob": strings.Repeat("x", MaxDataBytes)}},
	}

	for _, e := range events {
		if err := Write(ctx, e); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	q := db.New(testdb.Pool(t))
	actorText := pgtype.Text{String: actor, Valid: true}

	all, err := q.ListAuditEvents(ctx, db.ListAuditEventsParams{ActorID: actorText, Lim: 10})

	if err != nil {
		t.Fatal(err)
	}

	if len(all) != 4 || all[0].Action != "autovote.voted" {
		t.Fatalf("expected 4 events newest first, got %d", len(all))
	}

	if all[0].Data["truncated"] != true {
		t.Errorf("oversized data was stored as-is: %v", all[0].Data["truncated"])
	}

	payments, err := q.ListAuditEvents(ctx, db.ListAuditEventsParams{ActorID: actorText, ActionPrefix: pgtype.Text{String: "payment.", Valid: true}, Lim: 10})

	if err != nil || len(payments) != 1 || payments[0].Source != SourceStripe {
		t.Fatalf("action prefix filter: %v %d", err, len(payments))
	}

	failed, err := q.ListAuditEvents(ctx, db.ListAuditEventsParams{ActorID: actorText, FailedOnly: pgtype.Bool{Bool: true, Valid: true}, Lim: 10})

	if err != nil || len(failed) != 1 || failed[0].Status.Int32 != 403 {
		t.Fatalf("failed filter: %v %d", err, len(failed))
	}

	page, err := q.ListAuditEvents(ctx, db.ListAuditEventsParams{ActorID: actorText, Before: pgtype.Int8{Int64: all[1].ID, Valid: true}, Lim: 10})

	if err != nil || len(page) != 2 || page[0].ID != all[2].ID {
		t.Fatalf("before paging: %v %d", err, len(page))
	}

	target, err := q.ListAuditEvents(ctx, db.ListAuditEventsParams{ActorID: actorText, TargetID: pgtype.Text{String: "2", Valid: true}, Lim: 10})

	if err != nil || len(target) != 1 {
		t.Fatalf("target filter: %v %d", err, len(target))
	}
}
