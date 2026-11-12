// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"popplio/config"
	"popplio/db"
	"popplio/state"
	"popplio/testdb"
)

const discordTemplate = `{
	"code": "abc",
	"name": "Gaming Hub",
	"usage_count": 42,
	"source_guild_id": "123",
	"updated_at": "2026-11-10T12:00:00+00:00",
	"is_dirty": true,
	"serialized_source_guild": {
		"roles": [{"id": 0, "name": "@everyone", "color": 0}, {"id": 1, "name": "Mod", "color": 255}],
		"channels": [
			{"id": 2, "type": 4, "name": "General", "position": 0, "parent_id": null},
			{"id": 3, "type": 0, "name": "chat", "position": 0, "parent_id": 2}
		]
	}
}`

func TestSync(t *testing.T) {
	testdb.Pool(t)

	gone := false
	discord := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gone {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(discordTemplate))
	}))
	t.Cleanup(discord.Close)

	prev := state.Config
	state.Config = &config.Config{Meta: config.Meta{PopplioProxy: discord.URL}}
	t.Cleanup(func() { state.Config = prev })

	owner := testdb.ID(t)
	ctx := context.Background()
	q := db.New(state.Pool)

	id, err := q.InsertServerTemplate(ctx, db.InsertServerTemplateParams{
		Code:     "code-" + owner,
		Name:     "Old name",
		Short:    "A template",
		Tags:     []string{},
		Owner:    owner,
		Channels: []byte("[]"),
		Roles:    []byte("[]"),
	})
	if err != nil {
		t.Fatal(err)
	}
	testdb.Cleanup(t, "DELETE FROM server_templates WHERE owner = $1", owner)

	if _, err := Sync(ctx, id, "code-"+owner); err != nil {
		t.Fatalf("sync: %v", err)
	}

	row, err := q.GetServerTemplateByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if row.Name != "Gaming Hub" || row.UsageCount != 42 || !row.OutOfDate || row.Unavailable || !row.SyncedAt.Valid || !row.SourceUpdatedAt.Valid {
		t.Errorf("after sync: name=%q uses=%d outOfDate=%v unavailable=%v synced=%v", row.Name, row.UsageCount, row.OutOfDate, row.Unavailable, row.SyncedAt.Valid)
	}

	listed := func(ownerFilter string) bool {
		rows, err := q.GetServerTemplatesPaged(ctx, db.GetServerTemplatesPagedParams{Limit: 100, Tags: []string{}, Owner: ownerFilter})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.ID == id {
				if r.ChannelCount != 1 || r.RoleCount != 1 {
					t.Errorf("counts: channels=%d roles=%d, want 1 and 1", r.ChannelCount, r.RoleCount)
				}
				return true
			}
		}
		return false
	}

	if !listed("") {
		t.Error("a synced template should be listed")
	}

	gone = true

	if _, err := Sync(ctx, id, "code-"+owner); !errors.Is(err, ErrTemplateGone) {
		t.Fatalf("sync of a deleted template: err=%v, want ErrTemplateGone", err)
	}

	if listed("") {
		t.Error("an unavailable template is still listed in browse")
	}

	if !listed(owner) {
		t.Error("an unavailable template should still be listed for its owner")
	}

	gone = false

	if _, err := Sync(ctx, id, "code-"+owner); err != nil {
		t.Fatal(err)
	}

	if !listed("") {
		t.Error("a template that came back should be listed again")
	}
}
