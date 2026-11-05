package panel

import "testing"

func TestPanelEvent(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantOK     bool
		wantAction string
		wantTarget string
		wantToken  string
	}{
		{"read query", `{"BotQueue":{"login_token":"t"}}`, false, "", "", ""},
		{"list action", `{"UpdatePromoCodes":{"login_token":"t","action":"List"}}`, false, "", "", ""},
		{"get action", `{"UpdateStaffMembers":{"login_token":"t","action":{"GetMember":{"user_id":"1"}}}}`, false, "", "", ""},
		{"create", `{"UpdatePromoCodes":{"login_token":"t","action":{"Create":{"code":"X","products":["premium"]}}}}`, true, "panel.UpdatePromoCodes.Create", "", "t"},
		{"set active", `{"UpdatePromoCodes":{"login_token":"t","action":{"SetActive":{"id":"abc","active":false}}}}`, true, "panel.UpdatePromoCodes.SetActive", "abc", "t"},
		{"execute rpc", `{"ExecuteRpc":{"login_token":"t","target_type":"Bot","method":{"Approve":{"target_id":"42","reason":"ok"}}}}`, true, "panel.ExecuteRpc.Approve", "42", "t"},
		{"proxy get", `{"PopplioStaff":{"login_token":"t","path":"/staff/auto-vote","method":"GET","body":""}}`, false, "", "", ""},
		{"proxy patch", `{"PopplioStaff":{"login_token":"t","path":"/staff/apps/1","method":"PATCH","body":"{}"}}`, true, "panel.PopplioStaff", "", "t"},
		{"login", `{"Authorize":{"version":1,"action":{"Begin":{"code":"secret"}}}}`, true, "panel.Authorize.Begin", "", ""},
		{"not json", `nope`, false, "", "", ""},
	}

	for _, c := range cases {
		event, token, ok := panelEvent([]byte(c.body))

		if ok != c.wantOK {
			t.Errorf("%s: ok = %v, want %v", c.name, ok, c.wantOK)
			continue
		}

		if !ok {
			continue
		}

		if event.Action != c.wantAction || event.TargetID != c.wantTarget || token != c.wantToken {
			t.Errorf("%s: got %q %q %q, want %q %q %q", c.name, event.Action, event.TargetID, token, c.wantAction, c.wantTarget, c.wantToken)
		}

		if _, leaked := event.Data["login_token"]; leaked {
			t.Errorf("%s: login_token was stored", c.name)
		}

		if c.name == "login" && event.Data != nil {
			t.Errorf("login payload was stored: %v", event.Data)
		}
	}
}

func TestScrubRemovesNestedSecrets(t *testing.T) {
	got := scrub(map[string]any{
		"login_token": "a",
		"action":      map[string]any{"Verify": map[string]any{"otp": "123456", "keep": "yes"}},
		"list":        []any{map[string]any{"secret": "s", "name": "n"}},
	}).(map[string]any)

	if _, ok := got["login_token"]; ok {
		t.Error("login_token kept")
	}

	verify := got["action"].(map[string]any)["Verify"].(map[string]any)

	if _, ok := verify["otp"]; ok || verify["keep"] != "yes" {
		t.Errorf("nested scrub wrong: %v", verify)
	}

	item := got["list"].([]any)[0].(map[string]any)

	if _, ok := item["secret"]; ok || item["name"] != "n" {
		t.Errorf("list scrub wrong: %v", item)
	}
}
