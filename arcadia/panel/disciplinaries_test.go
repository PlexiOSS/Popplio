// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"
	ptypes "popplio/types"
)

func seedDisciplinaryType(t *testing.T, selfAssignable, needsApproval bool, maxExpiry string, permLimits []string) string {
	t.Helper()

	id := "test_" + impls.GenRandom(10)
	ctx := context.Background()

	var maxExpiryArg any

	if maxExpiry != "" {
		maxExpiryArg = maxExpiry
	}

	_, err := state.Pool.Exec(ctx,
		`INSERT INTO staff_disciplinary_types (id, name, description, self_assignable, perm_limits, additory, needs_approval, max_expiry)
		VALUES ($1, $2, 'test type', $3, $4, false, $5, $6::interval)`,
		id, "Type "+id, selfAssignable, permLimits, needsApproval, maxExpiryArg)

	if err != nil {
		t.Fatalf("seed type: %v", err)
	}

	t.Cleanup(func() {
		state.Pool.Exec(ctx, "DELETE FROM staff_disciplinary WHERE type = $1", id)
		state.Pool.Exec(ctx, "DELETE FROM staff_disciplinary_types WHERE id = $1", id)
	})

	return id
}

func disciplinaryQuery(t *testing.T, token string, action map[string]any) (int, types.StaffDisciplinary, string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"UpdateDisciplinaries": map[string]any{"login_token": token, "action": action},
	})

	if err != nil {
		t.Fatal(err)
	}

	rec := post(t, string(body))

	var d types.StaffDisciplinary

	if rec.Code == http.StatusOK {
		_ = json.Unmarshal(rec.Body.Bytes(), &d)
	}

	return rec.Code, d, rec.Body.String()
}

func TestDisciplinaryWorkflow(t *testing.T) {
	dbOrSkip(t)

	var notified []string
	prev := notifyDisciplinary
	notifyDisciplinary = func(userID, title, message string, _ ptypes.AlertType) {
		notified = append(notified, userID+": "+title)
	}
	t.Cleanup(func() { notifyDisciplinary = prev })

	senior := seedStaffAt(t, []string{"manage_disciplinaries"}, "active", 10)
	mid := seedStaffAt(t, []string{"manage_disciplinaries"}, "active", 30)
	junior := seedStaffAt(t, []string{"review_entities"}, "active", 50)

	breakType := seedDisciplinaryType(t, true, false, "7 days", []string{})
	hiatusType := seedDisciplinaryType(t, true, true, "", []string{"marker_disciplinary"})
	warningType := seedDisciplinaryType(t, false, true, "", []string{})

	t.Run("a self-assignable leave starts straight away and removes permissions", func(t *testing.T) {
		code, d, body := disciplinaryQuery(t, junior.Token, map[string]any{
			"Request": map[string]any{"type": breakType, "title": "", "description": "Holiday", "expiry": 3600},
		})

		if code != http.StatusOK || d.State != "active" || d.EndsAt == nil {
			t.Fatalf("request: %d %s", code, body)
		}

		resolved, err := perms.StaffPerms(context.Background(), junior.UserID)

		if err != nil {
			t.Fatal(err)
		}

		if resolved.Has(perms.StaffReviewEntities) {
			t.Fatalf("still has review_entities during leave: %v", resolved.Strings())
		}

		code, _, _ = disciplinaryQuery(t, junior.Token, map[string]any{
			"Request": map[string]any{"type": breakType, "title": "", "description": "", "expiry": 3600},
		})

		if code != http.StatusConflict {
			t.Fatalf("second request = %d, want 409", code)
		}

		code, d, body = disciplinaryQuery(t, junior.Token, map[string]any{"Revoke": map[string]any{"id": d.ID, "note": "Back early"}})

		if code != http.StatusOK || d.State != "revoked" {
			t.Fatalf("end own leave: %d %s", code, body)
		}
	})

	t.Run("a leave needing approval waits, then applies with no expiry", func(t *testing.T) {
		code, d, body := disciplinaryQuery(t, junior.Token, map[string]any{
			"Request": map[string]any{"type": hiatusType, "title": "Exams", "description": "", "expiry": nil},
		})

		if code != http.StatusOK || d.State != "pending" {
			t.Fatalf("request: %d %s", code, body)
		}

		active, err := impls.GetStaffDisciplinaries(context.Background(), junior.UserID, true)

		if err != nil || len(active) != 0 {
			t.Fatalf("pending counted as active: %v %v", active, err)
		}

		if code, _, _ := disciplinaryQuery(t, junior.Token, map[string]any{"Approve": map[string]any{"id": d.ID}}); code != http.StatusForbidden {
			t.Fatalf("self approve = %d, want 403", code)
		}

		code, d, body = disciplinaryQuery(t, mid.Token, map[string]any{"Approve": map[string]any{"id": d.ID, "note": "Good luck"}})

		if code != http.StatusOK || d.State != "active" || d.StartsAt == nil {
			t.Fatalf("approve: %d %s", code, body)
		}

		active, err = impls.GetStaffDisciplinaries(context.Background(), junior.UserID, true)

		if err != nil || len(active) != 1 {
			t.Fatalf("permanent disciplinary not active: %v %v", active, err)
		}
	})

	t.Run("issuing needs the permission and a higher rank", func(t *testing.T) {
		issue := func(token, userID, typeID string, expiry any) (int, types.StaffDisciplinary, string) {
			return disciplinaryQuery(t, token, map[string]any{
				"Issue": map[string]any{"user_id": userID, "type": typeID, "title": "Test", "description": "Reason", "expiry": expiry},
			})
		}

		if code, _, _ := issue(junior.Token, mid.UserID, warningType, nil); code != http.StatusForbidden {
			t.Fatalf("issue without perm = %d", code)
		}

		if code, _, _ := issue(mid.Token, senior.UserID, warningType, nil); code != http.StatusForbidden {
			t.Fatalf("issue to a higher rank = %d", code)
		}

		if code, _, _ := issue(mid.Token, junior.UserID, breakType, nil); code != http.StatusBadRequest {
			t.Fatalf("missing length = %d", code)
		}

		if code, _, _ := issue(mid.Token, junior.UserID, breakType, 30*24*3600); code != http.StatusBadRequest {
			t.Fatalf("length over the maximum = %d", code)
		}

		code, d, body := issue(mid.Token, junior.UserID, warningType, nil)

		if code != http.StatusOK || d.State != "pending" {
			t.Fatalf("issue: %d %s", code, body)
		}

		if code, _, _ := disciplinaryQuery(t, mid.Token, map[string]any{"Approve": map[string]any{"id": d.ID}}); code != http.StatusForbidden {
			t.Fatalf("issuer approving = %d, want 403", code)
		}

		code, d, body = disciplinaryQuery(t, senior.Token, map[string]any{"Deny": map[string]any{"id": d.ID, "note": "Not needed"}})

		if code != http.StatusOK || d.State != "denied" {
			t.Fatalf("deny: %d %s", code, body)
		}
	})

	t.Run("only managers can list other staff members", func(t *testing.T) {
		rec := post(t, fmt.Sprintf(`{"UpdateDisciplinaries":{"login_token":%q,"action":{"List":{"user_id":null,"state":null}}}}`, junior.Token))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("junior list all = %d", rec.Code)
		}

		rec = post(t, fmt.Sprintf(`{"UpdateDisciplinaries":{"login_token":%q,"action":{"List":{"user_id":%q,"state":null}}}}`, junior.Token, junior.UserID))

		if rec.Code != http.StatusOK {
			t.Fatalf("junior list own = %d %s", rec.Code, rec.Body.String())
		}
	})

	if len(notified) == 0 {
		t.Fatal("expected notifications for approvals, denials and issued disciplinaries")
	}
}
