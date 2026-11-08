// Copyright (C) 2026 NodeByte LTD

package put_site_prefs

import (
	"context"
	"testing"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"
)

func TestSitePrefsRoundTrip(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	user := testdb.ID(t)
	testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", user, "tok-"+user)
	testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", user)
	q := db.New(state.Pool)

	for _, prefs := range []map[string]any{
		{"accent": "indigo", "hiddenTags": []any{"economy"}},
		{"accent": "rose", "surface": "discord"},
	} {
		if _, err := q.UpsertSitePrefs(ctx, db.UpsertSitePrefsParams{UserID: user, Prefs: prefs}); err != nil {
			t.Fatal(err)
		}
	}

	row, err := q.GetSitePrefs(ctx, user)

	if err != nil {
		t.Fatal(err)
	}

	if row.Prefs["accent"] != "rose" || row.Prefs["surface"] != "discord" || row.Prefs["hiddenTags"] != nil {
		t.Fatalf("prefs = %v, want the second save to replace the first", row.Prefs)
	}

	testdb.Exec(t, "DELETE FROM users WHERE user_id = $1", user)

	if _, err := q.GetSitePrefs(ctx, user); err == nil {
		t.Fatal("prefs should be deleted with the user")
	}
}
