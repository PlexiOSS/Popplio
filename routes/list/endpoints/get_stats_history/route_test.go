// Copyright (C) 2026 NodeByte LTD

package get_stats_history

import (
	"context"
	"testing"
	"time"

	"popplio/testdb"
)

func TestBuildMonthlyHistory(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	voter, other, target, staff := testdb.ID(t), testdb.ID(t), testdb.ID(t), testdb.ID(t)

	for _, id := range []string{voter, other, staff} {
		testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", id, "tok-"+id)
		testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", id)
	}

	testdb.Exec(t, `INSERT INTO entity_votes (target_id, target_type, author, upvote, void, created_at) VALUES
		($1, 'bot', $2, true, true, '2019-03-15'),
		($1, 'bot', $3, true, true, '2019-03-20'),
		($1, 'bot', $2, false, true, '2019-03-21'),
		($1, 'bot', $2, true, true, '2019-05-02')`, target, voter, other)
	testdb.Cleanup(t, "DELETE FROM entity_votes WHERE target_id = $1", target)

	testdb.Exec(t, `INSERT INTO rpc_logs (user_id, method, data, state, created_at) VALUES
		($1, 'Approve', '{}', 'success', '2019-04-01'),
		($1, 'Approve', '{}', 'success', '2019-04-02'),
		($1, 'Deny', '{}', 'success', '2019-04-05'),
		($1, 'Approve', '{}', 'Whoa there! You need to test this bot first', '2019-04-06')`, staff)
	testdb.Cleanup(t, "DELETE FROM rpc_logs WHERE user_id = $1", staff)

	out, err := Build(ctx, time.Now())

	if err != nil {
		t.Fatal(err)
	}

	months := map[string]int{}

	for i, m := range out.Months {
		months[m.Month] = i

		if i > 0 && m.Month <= out.Months[i-1].Month {
			t.Fatalf("months out of order at %s", m.Month)
		}
	}

	if last := out.Months[len(out.Months)-1].Month; last != time.Now().UTC().Format("2006-01") {
		t.Fatalf("history ends at %s, want this month", last)
	}

	get := func(month string) (votes, voters, approved, denied *int64) {
		t.Helper()

		i, ok := months[month]

		if !ok {
			t.Fatalf("no entry for %s", month)
		}

		m := out.Months[i]

		return m.Votes, m.Voters, m.Approved, m.Denied
	}

	if votes, voters, approved, _ := get("2019-03"); votes == nil || *votes != 2 || *voters != 2 || approved != nil {
		t.Fatalf("2019-03: votes %v, voters %v, approved %v; want 2 upvotes from 2 people and no review record yet", votes, voters, approved)
	}

	if votes, _, approved, denied := get("2019-04"); votes == nil || *votes != 0 || approved == nil || *approved != 2 || *denied != 1 {
		t.Fatalf("2019-04: votes %v, approved %v, denied %v; want 0, 2, 1 (failed attempts don't count)", votes, approved, denied)
	}

	if votes, _, _, _ := get("2019-05"); votes == nil || *votes != 1 {
		t.Fatalf("2019-05: votes %v, want 1", votes)
	}

	if out.VotesAllTime < 3 || out.Approvals < 2 || out.Denials < 1 {
		t.Fatalf("totals = votes %d, approvals %d, denials %d", out.VotesAllTime, out.Approvals, out.Denials)
	}
}
