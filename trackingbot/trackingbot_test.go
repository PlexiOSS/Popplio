// Copyright (C) 2026 NodeByte LTD

package trackingbot

import (
	"context"
	"testing"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNextPhase(t *testing.T) {
	now := time.Now()
	at := func(d time.Duration) pgtype.Timestamptz {
		return pgtype.Timestamptz{Time: now.Add(-d), Valid: true}
	}

	cases := []struct {
		name   string
		since  pgtype.Timestamptz
		notice int16
		want   Phase
	}{
		{"present", pgtype.Timestamptz{}, noticeNone, PhaseOK},
		{"new and unwarned", at(time.Hour), noticeNone, PhaseWarn},
		{"warned, within grace", at(13 * 24 * time.Hour), noticeWarned, PhaseOK},
		{"warned, grace over", at(HideAfter), noticeWarned, PhaseHide},
		{"unwarned, grace over", at(HideAfter + time.Hour), noticeNone, PhaseHide},
		{"already hidden", at(30 * 24 * time.Hour), noticeHidden, PhaseOK},
	}

	for _, c := range cases {
		if got := NextPhase(c.since, c.notice, now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func seedServer(t *testing.T, listState string) string {
	t.Helper()
	ctx := context.Background()
	serverID := testdb.ID(t)

	var teamVanity, serverVanity, teamID string

	if err := state.Pool.QueryRow(ctx, "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'team', $2) RETURNING itag::text", serverID, "t"+serverID).Scan(&teamVanity); err != nil {
		t.Fatal(err)
	}

	testdb.Cleanup(t, "DELETE FROM vanity WHERE itag = $1::uuid", teamVanity)

	if err := state.Pool.QueryRow(ctx, "INSERT INTO teams (name, vanity_ref) VALUES ('Test team', $1::uuid) RETURNING id::text", teamVanity).Scan(&teamID); err != nil {
		t.Fatal(err)
	}

	testdb.Cleanup(t, "DELETE FROM teams WHERE id = $1::uuid", teamID)

	if err := state.Pool.QueryRow(ctx, "INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'server', $2) RETURNING itag::text", serverID, "s"+serverID).Scan(&serverVanity); err != nil {
		t.Fatal(err)
	}

	testdb.Cleanup(t, "DELETE FROM vanity WHERE itag = $1::uuid", serverVanity)
	testdb.Exec(t, `INSERT INTO servers (server_id, name, total_members, online_members, invite, team_owner, short, long, extra_links, vanity_ref, type, state)
		VALUES ($1, 'Test server', 10, 1, 'https://discord.gg/test', $2::uuid, 'short', 'long', '[]', $3::uuid, 'approved', $4)`, serverID, teamID, serverVanity, listState)
	testdb.Cleanup(t, "DELETE FROM servers WHERE server_id = $1", serverID)

	return serverID
}

type row struct {
	state   string
	hidden  pgtype.Text
	missing pgtype.Timestamptz
	notice  int16
}

func read(t *testing.T, serverID string) row {
	t.Helper()

	var r row

	if err := state.Pool.QueryRow(context.Background(),
		"SELECT state, tracking_bot_hidden_state, tracking_bot_missing_since, tracking_bot_notice FROM servers WHERE server_id = $1", serverID,
	).Scan(&r.state, &r.hidden, &r.missing, &r.notice); err != nil {
		t.Fatal(err)
	}

	return r
}

func TestSweepHidesAndRestores(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	serverID := seedServer(t, "public")

	missing := func(_ context.Context, id string) (bool, error) { return id != serverID, nil }
	present := func(context.Context, string) (bool, error) { return true, nil }

	if err := Sweep(ctx, missing); err != nil {
		t.Fatal(err)
	}

	r := read(t, serverID)
	if !r.missing.Valid || r.notice != noticeWarned || r.state != "public" {
		t.Fatalf("after first sweep: %+v", r)
	}

	testdb.Exec(t, "UPDATE servers SET tracking_bot_missing_since = NOW() - interval '15 days' WHERE server_id = $1", serverID)

	if err := Sweep(ctx, missing); err != nil {
		t.Fatal(err)
	}

	r = read(t, serverID)
	if r.state != "unlisted" || r.hidden.String != "public" || r.notice != noticeHidden {
		t.Fatalf("after grace period: %+v", r)
	}

	if err := db.New(state.Pool).UpdateServerSettings(ctx, db.UpdateServerSettingsParams{
		Short: "short", Long: "long", ExtraLinks: []byte("[]"), State: "private", Tags: []string{}, ServerID: serverID,
	}); err != nil {
		t.Fatal(err)
	}

	r = read(t, serverID)
	if r.state != "unlisted" || r.hidden.String != "private" {
		t.Fatalf("owner edit while hidden: %+v", r)
	}

	if err := Sweep(ctx, present); err != nil {
		t.Fatal(err)
	}

	r = read(t, serverID)
	if r.state != "private" || r.hidden.Valid || r.missing.Valid || r.notice != noticeNone {
		t.Fatalf("after bot returned: %+v", r)
	}
}

func TestPrivateListingIsNotChanged(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	serverID := seedServer(t, "private")

	testdb.Exec(t, "UPDATE servers SET tracking_bot_missing_since = NOW() - interval '20 days', tracking_bot_notice = 1 WHERE server_id = $1", serverID)

	if err := Sweep(ctx, func(_ context.Context, id string) (bool, error) { return id != serverID, nil }); err != nil {
		t.Fatal(err)
	}

	r := read(t, serverID)
	if r.state != "private" || r.hidden.Valid || r.notice != noticeHidden {
		t.Fatalf("private listing: %+v", r)
	}

	if err := Present(ctx, serverID); err != nil {
		t.Fatal(err)
	}

	if r = read(t, serverID); r.state != "private" || r.missing.Valid {
		t.Fatalf("private listing after return: %+v", r)
	}
}
