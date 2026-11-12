// Copyright (C) 2026 NodeByte LTD

package reports

import (
	"context"
	"testing"

	"popplio/db"
	"popplio/testdb"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestListReportsPage(t *testing.T) {
	q := db.New(testdb.Pool(t))
	ctx := context.Background()
	bot := testdb.ID(t)

	testdb.Exec(t, `INSERT INTO internal_user_cache__discord (id, username, display_name, avatar, bot) VALUES ($1, 'testbot', 'Test Bot', 'https://cdn.discordapp.com/avatars/1/a.png', true)`, bot)
	testdb.Cleanup(t, "DELETE FROM internal_user_cache__discord WHERE id = $1", bot)
	testdb.Cleanup(t, "DELETE FROM reports WHERE target_id = $1", bot)

	for range 25 {
		testdb.Exec(t, `INSERT INTO reports (target_type, target_id, reporter_id, reason, description, status) VALUES ('bot', $1, $2, 'spam', 'test', 'dismissed')`, bot, testdb.ID(t))
	}

	status := pgtype.Text{String: "dismissed", Valid: true}

	first, err := q.ListReportsPage(ctx, db.ListReportsPageParams{Status: status, Limit: 20, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}

	if len(first) != 20 || first[0].Total < 25 {
		t.Fatalf("first page: %d rows, total %d", len(first), first[0].Total)
	}

	found := false

	for offset := int32(0); offset < first[0].Total; offset += 20 {
		rows, err := q.ListReportsPage(ctx, db.ListReportsPageParams{Status: status, Limit: 20, Offset: offset})
		if err != nil {
			t.Fatal(err)
		}

		for _, r := range rows {
			if r.TargetID == bot {
				found = true

				if r.TargetName != "Test Bot" || r.TargetAvatar == "" {
					t.Fatalf("target = %q %q, want the cached display name and avatar", r.TargetName, r.TargetAvatar)
				}
			}
		}
	}

	if !found {
		t.Fatal("seeded reports weren't listed")
	}
}
