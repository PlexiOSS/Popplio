package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dbOrSkip(t *testing.T) {
	t.Helper()

	if state.Pool != nil {
		return
	}

	url := os.Getenv("ARCADIA_TEST_DATABASE_URL")

	if url == "" {
		t.Skip("ARCADIA_TEST_DATABASE_URL is not set; skipping integration tests")
	}

	pool, err := pgxpool.New(context.Background(), url)

	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	state.Pool = pool
}

type fixture struct {
	UserID     string
	Token      string
	PositionID string
}

func seedStaff(t *testing.T, perms []string, sessionState string) fixture {
	t.Helper()
	dbOrSkip(t)

	ctx := context.Background()

	f := fixture{
		UserID: fmt.Sprintf("9%014d", len(t.Name())*7919+len(perms)),
		Token:  impls.GenRandom(64),
	}

	f.UserID = fmt.Sprintf("test_%s", impls.GenRandom(12))

	cleanup := func() {
		state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE user_id = $1", f.UserID)
		state.Pool.Exec(ctx, "DELETE FROM staff_members WHERE user_id = $1", f.UserID)
		state.Pool.Exec(ctx, "DELETE FROM rpc_logs WHERE user_id = $1", f.UserID)
		state.Pool.Exec(ctx, "DELETE FROM users WHERE user_id = $1", f.UserID)

		if f.PositionID != "" {
			state.Pool.Exec(ctx, "DELETE FROM staff_positions WHERE id = $1", f.PositionID)
		}
	}

	cleanup()
	t.Cleanup(cleanup)

	_, err := state.Pool.Exec(ctx, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", f.UserID, impls.GenRandom(64))

	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	err = state.Pool.QueryRow(ctx,
		"INSERT INTO staff_positions (name, role_id, perms, index) VALUES ($1, $2, $3, $4) RETURNING id::text",
		"test_"+impls.GenRandom(8), "0", perms, 50,
	).Scan(&f.PositionID)

	if err != nil {
		t.Fatalf("seed position: %v", err)
	}

	_, err = state.Pool.Exec(ctx,
		"INSERT INTO staff_members (user_id, positions) VALUES ($1, ARRAY[$2::uuid])",
		f.UserID, f.PositionID)

	if err != nil {
		t.Fatalf("seed staff member: %v", err)
	}

	_, err = state.Pool.Exec(ctx,
		"INSERT INTO staffpanel__authchain (user_id, token, popplio_token, state) VALUES ($1, $2, $3, $4)",
		f.UserID, f.Token, impls.GenRandom(64), sessionState)

	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	return f
}

func post(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	handler(t).ServeHTTP(rec, req)

	return rec
}

func TestAuthValidators(t *testing.T) {
	dbOrSkip(t)

	ctx := context.Background()

	t.Run("unknown token is identityExpired", func(t *testing.T) {
		_, err := impls.CheckAuthInsecure(ctx, "nope-"+impls.GenRandom(20))

		if err == nil || err.Error() != "identityExpired" {
			t.Errorf("err = %v, want identityExpired", err)
		}
	})

	t.Run("pending session fails CheckAuth but passes insecure", func(t *testing.T) {
		f := seedStaff(t, []string{"review_entities"}, "pending")

		data, err := impls.CheckAuthInsecure(ctx, f.Token)

		if err != nil {
			t.Fatalf("CheckAuthInsecure: %v", err)
		}

		if data.UserID != f.UserID || data.State != "pending" {
			t.Errorf("auth data = %+v", data)
		}

		if _, err := impls.CheckAuth(ctx, f.Token); err == nil || err.Error() != "sessionNotActive" {
			t.Errorf("CheckAuth err = %v, want sessionNotActive", err)
		}
	})

	t.Run("active session passes both", func(t *testing.T) {
		f := seedStaff(t, []string{"review_entities"}, "active")

		if _, err := impls.CheckAuth(ctx, f.Token); err != nil {
			t.Errorf("CheckAuth: %v", err)
		}
	})

	t.Run("staff member with no positions is identityExpired", func(t *testing.T) {
		f := seedStaff(t, []string{}, "active")

		if _, err := state.Pool.Exec(ctx, "UPDATE staff_members SET positions = '{}' WHERE user_id = $1", f.UserID); err != nil {
			t.Fatalf("blank positions: %v", err)
		}

		if _, err := impls.CheckAuth(ctx, f.Token); err == nil || err.Error() != "identityExpired" {
			t.Errorf("err = %v, want identityExpired", err)
		}
	})

	t.Run("session GC removes stale pending sessions", func(t *testing.T) {
		f := seedStaff(t, []string{}, "pending")

		_, err := state.Pool.Exec(ctx,
			"UPDATE staffpanel__authchain SET created_at = NOW() - INTERVAL '10 minutes' WHERE token = $1", f.Token)

		if err != nil {
			t.Fatalf("age session: %v", err)
		}

		impls.CheckAuthInsecure(ctx, "trigger-gc")

		var count int64

		err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM staffpanel__authchain WHERE token = $1", f.Token).Scan(&count)

		if err != nil {
			t.Fatalf("count: %v", err)
		}

		if count != 0 {
			t.Error("stale pending session survived the GC")
		}
	})
}

func TestAuthFailureShape(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"BaseAnalytics":{"login_token":"definitely-not-a-token"}}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}

	if got := rec.Body.String(); got != "identityExpired" {
		t.Errorf("body = %q, want identityExpired", got)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
}

func TestLogoutReturnsRowCountAsText(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"Authorize":{"version":5,"action":{"Logout":{"login_token":%q}}}}`, f.Token))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	if got := rec.Body.String(); got != "1" {
		t.Errorf("body = %q, want \"1\"", got)
	}

	rec = post(t, fmt.Sprintf(`{"Authorize":{"version":5,"action":{"Logout":{"login_token":%q}}}}`, f.Token))

	if got := rec.Body.String(); got != "0" {
		t.Errorf("second logout body = %q, want \"0\"", got)
	}
}

func TestHelloChecksAuthBeforeVersion(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"Hello":{"login_token":"bad","version":999}}`)

	if got := rec.Body.String(); got != "identityExpired" {
		t.Errorf("body = %q, want identityExpired (auth is checked first)", got)
	}
}

func TestAuthorizeVersionGate(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"Authorize":{"version":4,"action":{"Begin":{"scope":"s","redirect_url":"r"}}}}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}

	if got := rec.Body.String(); got != "Invalid version" {
		t.Errorf("body = %q, want Invalid version", got)
	}
}

func TestPermissionGates(t *testing.T) {
	dbOrSkip(t)

	cases := []struct {
		name       string
		perm       string
		body       func(token string) string
		wantDenied string
	}{
		{
			name:       "GetRpcLogEntries",
			perm:       "view_audit_logs",
			body:       func(tok string) string { return fmt.Sprintf(`{"GetRpcLogEntries":{"login_token":%q}}`, tok) },
			wantDenied: "You do not have permission to view rpc logs [view_audit_logs]",
		},
		{
			name: "UpdateShopCoupons/List",
			perm: "view_shop",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateShopCoupons":{"login_token":%q,"action":"List"}}`, tok)
			},
			wantDenied: "You do not have permission to list shop coupons [view_shop]",
		},
		{
			name: "UpdateVoteCreditTiers/DeleteTier",
			perm: "manage_shop",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateVoteCreditTiers":{"login_token":%q,"action":{"DeleteTier":{"id":"nope"}}}}`, tok)
			},
			wantDenied: "You do not have permission to delete vote credit tiers [manage_shop]",
		},
		{
			name: "UpdateBotWhitelist/Delete",
			perm: "manage_bot_whitelist",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateBotWhitelist":{"login_token":%q,"action":{"Delete":{"bot_id":"nope"}}}}`, tok)
			},
			wantDenied: "You do not have permission to delete bot whitelist entries (bot_whitelist.delete)",
		},
		{
			name: "UpdateBlog/DeleteEntry",
			perm: "manage_blog",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateBlog":{"login_token":%q,"action":{"DeleteEntry":{"itag":"00000000-0000-0000-0000-000000000000"}}}}`, tok)
			},
			wantDenied: "You do not have permission to delete blog entries [manage_blog]",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name+"/denied", func(t *testing.T) {
			f := seedStaff(t, []string{"view_apps"}, "active")

			rec := post(t, tt.body(f.Token))

			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
			}

			if got := rec.Body.String(); got != tt.wantDenied {
				t.Errorf("body = %q\nwant %q", got, tt.wantDenied)
			}
		})

		t.Run(tt.name+"/allowed", func(t *testing.T) {
			f := seedStaff(t, []string{tt.perm}, "active")

			rec := post(t, tt.body(f.Token))

			if rec.Code == http.StatusForbidden && rec.Body.String() == tt.wantDenied {
				t.Errorf("permission %q was granted but the request was still denied", tt.perm)
			}
		})
	}
}

// A third tier landing on an already-doubly-occupied position used to 500
// with a raw Postgres constraint violation (CONFORMANCE.md #16). All three
// creates must now succeed, each existing tier pushed down by exactly one
// per insert, ending with every id at a distinct position.
func TestVoteCreditTierDedupLoop(t *testing.T) {
	f := seedStaff(t, []string{"manage_shop"}, "active")

	ctx := context.Background()

	create := func(id string) *httptest.ResponseRecorder {
		return post(t, fmt.Sprintf(
			`{"UpdateVoteCreditTiers":{"login_token":%q,"action":{"CreateTier":{"id":%q,"target_type":"bot","position":1,"cents":0.5,"votes":100}}}}`,
			f.Token, id))
	}

	ids := []string{"tier_a_" + impls.GenRandom(6), "tier_b_" + impls.GenRandom(6), "tier_c_" + impls.GenRandom(6)}

	t.Cleanup(func() {
		for _, id := range ids {
			state.Pool.Exec(ctx, "DELETE FROM vote_credit_tiers WHERE id = $1", id)
		}
	})

	for _, id := range ids[:2] {
		if rec := create(id); rec.Code != http.StatusNoContent {
			t.Fatalf("create %s: status = %d, body = %s", id, rec.Code, rec.Body.String())
		}
	}

	positions := tierPositions(t, ids[:2])

	if positions[ids[0]] != 2 || positions[ids[1]] != 1 {
		t.Errorf("after two creates positions = %v, want the first tier pushed to 2 and the second at 1", positions)
	}

	rec := create(ids[2])

	if rec.Code != http.StatusNoContent {
		t.Fatalf("third create: status = %d, want 204; body = %s", rec.Code, rec.Body.String())
	}

	positions = tierPositions(t, ids)

	if positions[ids[0]] != 3 || positions[ids[1]] != 2 || positions[ids[2]] != 1 {
		t.Errorf("after three creates positions = %v, want the first two pushed down again and the third at 1", positions)
	}
}

func tierPositions(t *testing.T, ids []string) map[string]int32 {
	t.Helper()

	rows, err := state.Pool.Query(context.Background(),
		"SELECT id, position FROM vote_credit_tiers WHERE id = ANY($1)", ids)

	if err != nil {
		t.Fatalf("query positions: %v", err)
	}

	defer rows.Close()

	out := map[string]int32{}

	for rows.Next() {
		var (
			id  string
			pos int32
		)

		if err := rows.Scan(&id, &pos); err != nil {
			t.Fatalf("scan: %v", err)
		}

		out[id] = pos
	}

	return out
}

// A nil max_uses/reuse_wait_duration/expiry means "unlimited"/"never
// expires" per the DTO's own doc comments, and must be accepted — see
// CONFORMANCE.md #2. A present-but-non-positive value is still rejected.
func TestShopCouponNullMeansUnconstrained(t *testing.T) {
	f := seedStaff(t, []string{"manage_shop"}, "active")

	body := fmt.Sprintf(`{"UpdateShopCoupons":{"login_token":%q,"action":{"Create":{"id":"x","code":"c","public":true,"max_uses":null,"reuse_wait_duration":null,"expiry":null,"applicable_items":[],"cents":null,"requirements":[],"allowed_users":[],"usable":true,"target_types":[]}}}}`, f.Token)

	rec := post(t, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestShopCouponNonPositiveMaxUsesIsRejected(t *testing.T) {
	f := seedStaff(t, []string{"manage_shop"}, "active")

	body := fmt.Sprintf(`{"UpdateShopCoupons":{"login_token":%q,"action":{"Create":{"id":"x","code":"c","public":true,"max_uses":0,"reuse_wait_duration":1,"expiry":1,"applicable_items":[],"cents":null,"requirements":[],"allowed_users":[],"usable":true,"target_types":[]}}}}`, f.Token)

	rec := post(t, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}

	if got := rec.Body.String(); got != "Max uses must be greater than 0" {
		t.Errorf("body = %q, want %q", got, "Max uses must be greater than 0")
	}
}

func TestListPositionsIsOpenToAllStaff(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"UpdateStaffPositions":{"login_token":%q,"action":"ListPositions"}}`, f.Token))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var positions []types.StaffPosition

	if err := json.Unmarshal(rec.Body.Bytes(), &positions); err != nil {
		t.Fatalf("response is not a StaffPosition array: %v\nbody: %s", err, rec.Body.String())
	}

	var found bool

	for _, p := range positions {
		if p.ID == f.PositionID {
			found = true

			if p.CorrespondingRoles == nil {
				t.Error("corresponding_roles decoded as null, want []")
			}
		}
	}

	if !found {
		t.Error("the seeded position is missing from ListPositions")
	}
}

func TestBaseAnalytics(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"BaseAnalytics":{"login_token":%q}}`, f.Token))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var analytics types.BaseAnalytics

	if err := json.Unmarshal(rec.Body.Bytes(), &analytics); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, rec.Body.String())
	}

	if analytics.ChangelogsCount != 0 {
		t.Errorf("changelogs_count = %d, want 0 (hardcoded)", analytics.ChangelogsCount)
	}

	if analytics.TotalUsers < 1 {
		t.Errorf("total_users = %d, want at least the seeded user", analytics.TotalUsers)
	}

	if analytics.BotCounts == nil || analytics.ServerCounts == nil || analytics.TicketCounts == nil {
		t.Error("a count map decoded as null")
	}
}

func TestChangelogRequiresAuth(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"UpdateChangelog":{"login_token":"not-a-real-token","action":"ListEntries"}}`)

	if got := rec.Body.String(); got != "identityExpired" {
		t.Errorf("body = %q, want identityExpired (no longer the hard-coded stub message)", got)
	}
}

func TestChangelogCreateRequiresPerm(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"UpdateChangelog":{"login_token":%q,"action":{"CreateEntry":{"project":"popplio","version":"9.9.9","extra_description":"","prerelease":false,"published":false,"added":[],"updated":[],"removed":[]}}}}`, f.Token))

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}

	if got := rec.Body.String(); got != "You do not have permission to create changelog entries [manage_changelog]" {
		t.Errorf("body = %q", got)
	}
}

func TestChangelogCreateListDelete(t *testing.T) {
	f := seedStaff(t, []string{"manage_changelog"}, "active")

	create := post(t, fmt.Sprintf(`{"UpdateChangelog":{"login_token":%q,"action":{"CreateEntry":{"project":"omniplex","version":"test-%s","extra_description":"integration test entry","prerelease":true,"published":true,"added":["a thing"],"updated":[],"removed":[]}}}}`, f.Token, impls.GenRandom(8)))

	if create.Code != http.StatusNoContent {
		t.Fatalf("create status = %d, want 204: %s", create.Code, create.Body.String())
	}

	list := post(t, fmt.Sprintf(`{"UpdateChangelog":{"login_token":%q,"action":"ListEntries"}}`, f.Token))

	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", list.Code, list.Body.String())
	}

	var entries []types.ChangelogEntry

	if err := json.Unmarshal(list.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least the entry just created")
	}

	itag := entries[0].Itag

	del := post(t, fmt.Sprintf(`{"UpdateChangelog":{"login_token":%q,"action":{"DeleteEntry":{"itag":%q}}}}`, f.Token, itag))

	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204: %s", del.Code, del.Body.String())
	}
}
