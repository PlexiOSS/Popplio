// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"testing"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"
)

func TestSearchFiltersCantRevealHiddenStats(t *testing.T) {
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
	testdb.Exec(t, `INSERT INTO bots (bot_id, client_id, tags, prefix, short, long, extra_links, invite, vanity_ref, owner, type, servers, shards)
		VALUES ($1, $1, '{}', '!', 'short', 'long', '[]', 'https://example.com', $2::uuid, $3, 'approved', 5000, 4)`, botID, vanity, owner)
	testdb.Cleanup(t, "DELETE FROM bots WHERE bot_id = $1", botID)
	q := db.New(state.Pool)

	search := func(params db.SearchBotsPublicParams) bool {
		t.Helper()
		params.Query = botID
		params.Pattern = "%" + botID + "%"
		params.Prefix = botID + "%"
		params.TagMode = "@>"
		params.Tags = []string{}
		params.Lim = 10

		rows, err := q.SearchBotsPublic(ctx, params)

		if err != nil {
			t.Fatal(err)
		}

		return len(rows) == 1
	}

	byServers := db.SearchBotsPublicParams{ServersFrom: 4000, ServersTo: 6000}
	byShards := db.SearchBotsPublicParams{ShardsFrom: 3, ShardsTo: 5}

	if !search(byServers) || !search(byShards) || !search(db.SearchBotsPublicParams{}) {
		t.Fatal("a visible bot should match its range filters")
	}

	if err := q.SetBotHiddenStats(ctx, db.SetBotHiddenStatsParams{BotID: botID, HiddenStats: []string{"servers"}}); err != nil {
		t.Fatal(err)
	}

	if search(byServers) {
		t.Fatal("a server range filter revealed a hidden server count")
	}

	if !search(byShards) || !search(db.SearchBotsPublicParams{}) {
		t.Fatal("hiding servers should only affect the server filter")
	}

	hidden, err := q.GetBotHiddenStats(ctx, botID)

	if err != nil || len(hidden) != 1 || hidden[0] != "servers" {
		t.Fatalf("hidden stats = %v, %v", hidden, err)
	}
}
