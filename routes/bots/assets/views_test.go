// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"testing"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"
)

func TestRecentViewsAndCertification(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	owner, botID := testdb.ID(t), testdb.ID(t)
	testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", owner, "tok-"+owner)
	testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", owner)

	var vanity string

	if err := state.Pool.QueryRow(ctx,
		"INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'bot', $2) RETURNING itag::text", botID, "b"+botID,
	).Scan(&vanity); err != nil {
		t.Fatal(err)
	}

	testdb.Cleanup(t, "DELETE FROM vanity WHERE itag = $1::uuid", vanity)
	testdb.Exec(t, `INSERT INTO bots (bot_id, client_id, tags, prefix, short, long, extra_links, invite, vanity_ref, owner, type, clicks, unique_clicks)
		VALUES ($1, $1, '{}', '!', 'short', 'long', '[]', 'https://example.com', $2::uuid, $3, 'approved', 59000, ARRAY['a','b','c'])`, botID, vanity, owner)
	testdb.Cleanup(t, "DELETE FROM bots WHERE bot_id = $1", botID)
	testdb.Cleanup(t, "DELETE FROM entity_views_daily WHERE target_id = $1", botID)
	q := db.New(state.Pool)

	recent := func() int32 {
		t.Helper()

		views, err := q.GetEntityRecentViews(ctx, db.GetEntityRecentViewsParams{TargetType: "bot", TargetID: botID})

		if err != nil {
			t.Fatal(err)
		}

		return views
	}

	if recent() != 0 {
		t.Fatal("old totals shouldn't count as recent views")
	}

	certClicks := func() int32 {
		t.Helper()

		row, err := q.GetBotCertStats(ctx, botID)

		if err != nil {
			t.Fatal(err)
		}

		return row.UniqueClicks
	}

	if got := certClicks(); got != 3 {
		t.Fatalf("certification should keep the old count, got %d", got)
	}

	for range 2 {
		if err := q.RecordEntityView(ctx, db.RecordEntityViewParams{TargetType: "bot", TargetID: botID}); err != nil {
			t.Fatal(err)
		}
	}

	testdb.Exec(t, `INSERT INTO entity_views_daily (target_type, target_id, day, views) VALUES
		('bot', $1, CURRENT_DATE - 29, 18), ('bot', $1, CURRENT_DATE - 30, 100), ('bot', $1, CURRENT_DATE - 91, 7)`, botID)

	if got := recent(); got != 20 {
		t.Fatalf("recent views = %d, want 20", got)
	}

	if got := certClicks(); got != 20 {
		t.Fatalf("certification should use the higher count, got %d", got)
	}

	extras, err := q.GetBotListingExtras(ctx, botID)

	if err != nil || extras.Views30d != 20 {
		t.Fatalf("listing extras = %+v, %v", extras, err)
	}

	if err := q.PruneEntityViewsDaily(ctx); err != nil {
		t.Fatal(err)
	}

	var days int

	if err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM entity_views_daily WHERE target_id = $1", botID).Scan(&days); err != nil {
		t.Fatal(err)
	}

	if days != 3 {
		t.Fatalf("%d days left after pruning, want 3", days)
	}
}
