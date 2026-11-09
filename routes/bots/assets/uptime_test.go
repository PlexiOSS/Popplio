// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"testing"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"
)

func TestUptimePercent(t *testing.T) {
	if UptimePercent(0, 0) != nil {
		t.Error("no checks should be no data")
	}

	if UptimePercent(100, 0) != nil {
		t.Error("never seen online should be no data, not 0%")
	}

	if got := UptimePercent(4, 4); got == nil || *got != 100 {
		t.Errorf("all online = %v, want 100", got)
	}

	if got := UptimePercent(4, 1); got == nil || *got != 25 {
		t.Errorf("one of four online = %v, want 25", got)
	}
}

func TestRecentUptimeUsesTheLast30Days(t *testing.T) {
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
	testdb.Exec(t, `INSERT INTO bots (bot_id, client_id, tags, prefix, short, long, extra_links, invite, vanity_ref, owner, type, uptime, total_uptime)
		VALUES ($1, $1, '{}', '!', 'short', 'long', '[]', 'https://example.com', $2::uuid, $3, 'approved', 0, 29107)`, botID, vanity, owner)
	testdb.Cleanup(t, "DELETE FROM bots WHERE bot_id = $1", botID)
	q := db.New(state.Pool)

	if percent, checks := RecentUptime(ctx, botID); percent != nil || checks != 0 {
		t.Fatalf("before any check = %v, %d", percent, checks)
	}

	if err := q.RecordBotUptimeCheck(ctx, db.RecordBotUptimeCheckParams{BotID: botID, Online: false}); err != nil {
		t.Fatal(err)
	}

	if percent, checks := RecentUptime(ctx, botID); percent != nil || checks != 1 {
		t.Fatalf("after an offline check = %v, %d, want no data", percent, checks)
	}

	for range 3 {
		if err := q.RecordBotUptimeCheck(ctx, db.RecordBotUptimeCheckParams{BotID: botID, Online: true}); err != nil {
			t.Fatal(err)
		}
	}

	if percent, checks := RecentUptime(ctx, botID); percent == nil || *percent != 75 || checks != 4 {
		t.Fatalf("three of four online = %v, %d, want 75", percent, checks)
	}

	testdb.Exec(t, "INSERT INTO bot_uptime_daily (bot_id, day, checks, online) VALUES ($1, CURRENT_DATE - 30, 100, 0), ($1, CURRENT_DATE - 91, 5, 5)", botID)

	if percent, checks := RecentUptime(ctx, botID); percent == nil || *percent != 75 || checks != 4 {
		t.Fatalf("days outside the window counted: %v, %d", percent, checks)
	}

	if err := q.PruneBotUptimeDaily(ctx); err != nil {
		t.Fatal(err)
	}

	var left int

	if err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bot_uptime_daily WHERE bot_id = $1", botID).Scan(&left); err != nil {
		t.Fatal(err)
	}

	if left != 2 {
		t.Fatalf("%d days left after pruning, want 2", left)
	}

	row, err := q.GetBotRecentUptime(ctx, botID)

	if err != nil || row.Checks != 4 {
		t.Fatalf("recent row = %+v, %v", row, err)
	}
}
