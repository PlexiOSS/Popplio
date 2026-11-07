// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"popplio/apps"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
)

func positionQuery(t *testing.T, token string, action map[string]any) (int, string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"UpdateAppPositions": map[string]any{"login_token": token, "action": action},
	})

	if err != nil {
		t.Fatal(err)
	}

	rec := post(t, string(body))

	return rec.Code, rec.Body.String()
}

func TestAppPositions(t *testing.T) {
	dbOrSkip(t)

	editor := seedStaffAt(t, []string{"manage_app_positions"}, "active", 21)
	reviewer := seedStaffAt(t, []string{"view_apps"}, "active", 61)

	t.Run("the migration seeds the built-in positions with their behaviour", func(t *testing.T) {
		positions, err := apps.List(context.Background())

		if err != nil {
			t.Fatal(err)
		}

		byID := map[string]bool{}

		for _, p := range positions {
			byID[p.ID] = true

			if p.ID == "certification" && (p.ExtraLogic == nil || p.ReviewLogic == nil) {
				t.Fatal("certification lost its behaviour")
			}

			if p.ID == "banappeal" && !p.BannedOnly {
				t.Fatal("ban appeal isn't banned-only")
			}
		}

		for _, id := range []string{"staff", "dev", "partners", "resubmit", "banappeal", "certification", "server_certification"} {
			if !byID[id] {
				t.Fatalf("missing seeded position %s", id)
			}
		}
	})

	t.Run("listing needs view_apps and editing needs manage_app_positions", func(t *testing.T) {
		code, body := positionQuery(t, reviewer.Token, map[string]any{"List": map[string]any{}})

		if code != http.StatusOK {
			t.Fatalf("reviewer list = %d %s", code, body)
		}

		var list []types.AppPosition
		_ = json.Unmarshal([]byte(body), &list)

		for _, p := range list {
			if p.ID == "certification" && (p.Behaviour == nil || len(p.Behaviour.LockedQuestions) == 0) {
				t.Fatalf("certification should report its locked questions: %+v", p.Behaviour)
			}
		}

		if code, _ := positionQuery(t, reviewer.Token, map[string]any{"Delete": map[string]any{"id": "dev"}}); code != http.StatusForbidden {
			t.Fatalf("reviewer delete = %d", code)
		}
	})

	id := "test_" + strings.ToLower(impls.GenRandom(8))
	t.Cleanup(func() {
		state.Pool.Exec(context.Background(), "DELETE FROM app_positions WHERE id = $1", id)
	})

	input := func() map[string]any {
		return map[string]any{
			"id": id, "name": "Test Team", "info": "Join us", "tags": []string{"Test", "Test", " "},
			"questions": []map[string]any{
				{"id": "why", "question": "Why?", "paragraph": "", "placeholder": "", "short": true},
			},
			"hidden": false, "closed": false, "requires_main_server": false, "cooldown_secs": 3600,
			"approval_note": "", "help_url": "", "help_label": "",
		}
	}

	t.Run("creating validates and normalises the form", func(t *testing.T) {
		bad := input()
		bad["questions"] = []map[string]any{
			{"id": "a", "question": "One", "paragraph": "", "placeholder": "", "short": true},
			{"id": "a", "question": "Two", "paragraph": "", "placeholder": "", "short": true},
		}

		if code, body := positionQuery(t, editor.Token, map[string]any{"Create": bad}); code != http.StatusBadRequest {
			t.Fatalf("duplicate question IDs = %d %s", code, body)
		}

		code, body := positionQuery(t, editor.Token, map[string]any{"Create": input()})

		if code != http.StatusOK {
			t.Fatalf("create = %d %s", code, body)
		}

		var p types.AppPosition
		_ = json.Unmarshal([]byte(body), &p)

		if len(p.Tags) != 1 || p.Tags[0] != "Test" {
			t.Fatalf("tags = %v, want [Test]", p.Tags)
		}

		if code, _ := positionQuery(t, editor.Token, map[string]any{"Create": input()}); code != http.StatusConflict {
			t.Fatalf("duplicate create = %d", code)
		}

		found, err := apps.FindPosition(context.Background(), id)

		if err != nil || found == nil || found.Name != "Test Team" {
			t.Fatalf("FindPosition after create: %+v %v", found, err)
		}
	})

	t.Run("built-in positions keep their locked questions and can't be deleted", func(t *testing.T) {
		code, body := positionQuery(t, reviewer.Token, map[string]any{"List": map[string]any{}})

		if code != http.StatusOK {
			t.Fatal(body)
		}

		var list []types.AppPosition
		_ = json.Unmarshal([]byte(body), &list)

		var cert types.AppPosition

		for _, p := range list {
			if p.ID == "certification" {
				cert = p
			}
		}

		update := map[string]any{
			"id": cert.ID, "name": cert.Name, "info": cert.Info, "tags": cert.Tags,
			"questions": []map[string]any{
				{"id": "why", "question": "Why?", "paragraph": "", "placeholder": "", "short": true},
			},
			"hidden": cert.Hidden, "closed": cert.Closed, "requires_main_server": cert.RequiresMainServer,
			"cooldown_secs": cert.CooldownSecs, "approval_note": cert.ApprovalNote, "help_url": cert.HelpURL, "help_label": cert.HelpLabel,
		}

		if code, body := positionQuery(t, editor.Token, map[string]any{"Update": update}); code != http.StatusBadRequest {
			t.Fatalf("removing a locked question = %d %s", code, body)
		}

		if code, _ := positionQuery(t, editor.Token, map[string]any{"Delete": map[string]any{"id": "certification"}}); code != http.StatusForbidden {
			t.Fatalf("deleting a built-in = %d", code)
		}
	})

	t.Run("a position with pending applications can't be deleted", func(t *testing.T) {
		ctx := context.Background()
		appID := "test_" + impls.GenRandom(10)

		if _, err := state.Pool.Exec(ctx, `INSERT INTO apps (app_id, user_id, "position") VALUES ($1, $2, $3)`, appID, editor.UserID, id); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { state.Pool.Exec(ctx, "DELETE FROM apps WHERE app_id = $1", appID) })

		if code, body := positionQuery(t, editor.Token, map[string]any{"Delete": map[string]any{"id": id}}); code != http.StatusConflict {
			t.Fatalf("delete with pending = %d %s", code, body)
		}

		state.Pool.Exec(ctx, "DELETE FROM apps WHERE app_id = $1", appID)

		if code, body := positionQuery(t, editor.Token, map[string]any{"Delete": map[string]any{"id": id}}); code != http.StatusNoContent {
			t.Fatalf("delete = %d %s", code, fmt.Sprint(body))
		}
	})
}
