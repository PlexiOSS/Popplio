// Copyright (C) 2026 NodeByte LTD

package pagestyle

import (
	"context"
	"errors"
	"testing"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestReadableAccent(t *testing.T) {
	cases := map[string]bool{
		"#6366f1": true,
		"#10b981": true,
		"#ec4899": true,
		"#000000": false,
		"#ffffff": false,
		"#fef9c3": false,
		"#111827": false,
		"nope":    false,
	}

	for color, want := range cases {
		if got := ReadableAccent(color); got != want {
			t.Errorf("ReadableAccent(%s) = %v, want %v", color, got, want)
		}
	}
}

func seedBot(t *testing.T, premium bool) string {
	t.Helper()

	owner, botID := testdb.ID(t), testdb.ID(t)
	testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", owner, "tok-"+owner)
	testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", owner)

	var vanity string

	if err := state.Pool.QueryRow(context.Background(),
		"INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'bot', $2) RETURNING itag::text", botID, "b"+botID,
	).Scan(&vanity); err != nil {
		t.Fatal(err)
	}

	testdb.Cleanup(t, "DELETE FROM vanity WHERE itag = $1::uuid", vanity)
	testdb.Exec(t, `INSERT INTO bots (bot_id, client_id, tags, prefix, short, long, extra_links, invite, vanity_ref, owner, type, premium)
		VALUES ($1, $1, '{}', '!', 'short', 'long', '[]', 'https://example.com', $2::uuid, $3, 'approved', $4)`, botID, vanity, owner, premium)
	testdb.Cleanup(t, "DELETE FROM bots WHERE bot_id = $1", botID)
	testdb.Cleanup(t, "DELETE FROM entity_page_styles WHERE target_id = $1", botID)

	return botID
}

func settings() types.PageStyleUpdate {
	gradient, accent := "aurora", "#6366F1"
	return types.PageStyleUpdate{Gradient: &gradient, Accent: &accent, Dim: 60, Blur: 4, ImagePosition: "top"}
}

func TestUpdateNeedsPremium(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	free := seedBot(t, false)

	if err := Update(ctx, "bot", free, settings()); !errors.Is(err, ErrNotPremium) {
		t.Fatalf("update without premium = %v, want ErrNotPremium", err)
	}

	if err := SubmitImage(ctx, "bot", free, "0123456789abcdef"); !errors.Is(err, ErrNotPremium) {
		t.Fatalf("submit without premium = %v, want ErrNotPremium", err)
	}

	if err := Update(ctx, "bot", testdb.ID(t), settings()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update for a missing bot = %v, want ErrNotFound", err)
	}
}

func TestUpdateValidates(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	bot := seedBot(t, true)

	bad := settings()
	g := "rainbow"
	bad.Gradient = &g

	if err := Update(ctx, "bot", bot, bad); !errors.Is(err, ErrBadGradient) {
		t.Fatalf("unknown gradient = %v", err)
	}

	bad = settings()
	a := "#fefefe"
	bad.Accent = &a

	if err := Update(ctx, "bot", bot, bad); !errors.Is(err, ErrBadAccent) {
		t.Fatalf("unreadable accent = %v", err)
	}

	if err := Update(ctx, "bot", bot, settings()); err != nil {
		t.Fatal(err)
	}

	style, err := Public(ctx, "bot", bot)

	if err != nil || style == nil {
		t.Fatalf("public style = %v, %v", style, err)
	}

	if *style.Accent != "#6366f1" || *style.Gradient != "aurora" || style.Dim != 60 || style.ImagePosition != "top" {
		t.Fatalf("style = %+v", style)
	}
}

func TestImageNeedsApproval(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(state.Pool)
	bot := seedBot(t, true)

	if err := SubmitImage(ctx, "bot", bot, "AAAABBBBCCCCDDDD"); err != nil {
		t.Fatal(err)
	}

	if style, _ := Public(ctx, "bot", bot); style != nil && style.ImageVersion != nil {
		t.Fatal("a background was public before review")
	}

	managed, err := Managed(ctx, "bot", bot)

	if err != nil || managed.PendingVersion == nil || *managed.PendingVersion != "aaaabbbbccccdddd" {
		t.Fatalf("managed = %+v, %v; want the pending version", managed, err)
	}

	staff := pgtype.Text{String: "1", Valid: true}

	if n, err := q.RejectPageImage(ctx, db.RejectPageImageParams{Reason: pgtype.Text{String: "Too busy to read text over", Valid: true}, StaffID: staff, TargetType: "bot", TargetID: bot, Version: pgtype.Text{String: "aaaabbbbccccdddd", Valid: true}}); err != nil || n != 1 {
		t.Fatalf("reject = %d, %v", n, err)
	}

	managed, _ = Managed(ctx, "bot", bot)

	if managed.PendingVersion != nil || managed.RejectedReason == nil {
		t.Fatalf("after reject = %+v", managed)
	}

	if err := SubmitImage(ctx, "bot", bot, "1111222233334444"); err != nil {
		t.Fatal(err)
	}

	if n, _ := q.ApprovePageImage(ctx, db.ApprovePageImageParams{StaffID: staff, TargetType: "bot", TargetID: bot, Version: pgtype.Text{String: "aaaabbbbccccdddd", Valid: true}}); n != 0 {
		t.Fatal("approving a version that was replaced went through")
	}

	if n, err := q.ApprovePageImage(ctx, db.ApprovePageImageParams{StaffID: staff, TargetType: "bot", TargetID: bot, Version: pgtype.Text{String: "1111222233334444", Valid: true}}); err != nil || n != 1 {
		t.Fatalf("approve = %d, %v", n, err)
	}

	style, err := Public(ctx, "bot", bot)

	if err != nil || style == nil || style.ImageVersion == nil || *style.ImageVersion != "1111222233334444" {
		t.Fatalf("public after approval = %+v, %v", style, err)
	}

	testdb.Exec(t, "UPDATE bots SET premium = false WHERE bot_id = $1", bot)

	if style, _ := Public(ctx, "bot", bot); style != nil {
		t.Fatal("the style still showed after premium ended")
	}

	testdb.Exec(t, "UPDATE bots SET premium = true WHERE bot_id = $1", bot)

	remove := settings()
	remove.RemoveImage = true

	if err := Update(ctx, "bot", bot, remove); err != nil {
		t.Fatal(err)
	}

	if style, _ := Public(ctx, "bot", bot); style == nil || style.ImageVersion != nil {
		t.Fatalf("after removing the image = %+v", style)
	}
}

func TestPageStylePass(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	bot := seedBot(t, false)
	q := db.New(state.Pool)

	if err := q.ApplyPageStylePass(ctx, db.ApplyPageStylePassParams{TargetType: "bot", TargetID: bot, Hours: 720}); err != nil {
		t.Fatal(err)
	}

	if err := Update(ctx, "bot", bot, settings()); err != nil {
		t.Fatalf("update with a pass = %v", err)
	}

	if style, err := Public(ctx, "bot", bot); err != nil || style == nil {
		t.Fatalf("public style with a pass = %v, %v", style, err)
	}

	if err := q.ApplyPageStylePass(ctx, db.ApplyPageStylePassParams{TargetType: "bot", TargetID: bot, Hours: 720}); err != nil {
		t.Fatal(err)
	}

	managed, err := Managed(ctx, "bot", bot)

	if err != nil || !managed.Premium || managed.UnlockedUntil == nil {
		t.Fatalf("managed = %+v, %v", managed, err)
	}

	if left := time.Until(*managed.UnlockedUntil); left < 1430*time.Hour || left > 1441*time.Hour {
		t.Fatalf("a second pass should stack to about 60 days, got %v", left)
	}

	testdb.Exec(t, "UPDATE entity_page_styles SET unlocked_until = NOW() - INTERVAL '1 minute' WHERE target_id = $1", bot)

	if style, err := Public(ctx, "bot", bot); err != nil || style != nil {
		t.Fatalf("public style after the pass ran out = %v, %v", style, err)
	}

	if err := Update(ctx, "bot", bot, settings()); !errors.Is(err, ErrNotPremium) {
		t.Fatalf("update after the pass ran out = %v, want ErrNotPremium", err)
	}
}

func TestValidGradient(t *testing.T) {
	cases := map[string]bool{
		"aurora":                  true,
		"custom:#1e1b4b:#0f172a":  true,
		"custom:#1E1B4B:#0f172a":  false,
		"custom:#1e1b4b":          false,
		"custom:#1e1b4b:#0f172a;": false,
		"custom:red:blue":         false,
		"rainbow":                 false,
	}

	for id, want := range cases {
		if got := ValidGradient(id); got != want {
			t.Errorf("ValidGradient(%q) = %v, want %v", id, got, want)
		}
	}
}
