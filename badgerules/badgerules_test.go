// Copyright (C) 2026 NodeByte LTD

package badgerules

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		rule    *Rule
		targets []string
		ok      bool
	}{
		{"no rule", nil, []string{"user"}, true},
		{"unknown type", &Rule{Type: "nope"}, []string{"user"}, false},
		{"wrong target", &Rule{Type: "votes_cast", Value: 5}, []string{"bot"}, false},
		{"value too low", &Rule{Type: "votes_cast", Value: 0}, []string{"user"}, false},
		{"valid value", &Rule{Type: "votes_cast", Value: 5}, []string{"user"}, true},
		{"one supported target is enough", &Rule{Type: "listed_days", Value: 30}, []string{"user", "bot"}, true},
		{"role required", &Rule{Type: "has_role"}, []string{"user"}, false},
		{"role must be a snowflake", &Rule{Type: "has_role", RoleID: "abc"}, []string{"user"}, false},
		{"valid role", &Rule{Type: "has_role", RoleID: "758641373074423808"}, []string{"user"}, true},
		{"flag rule", &Rule{Type: "partner"}, []string{"bot"}, true},
	}

	for _, c := range cases {
		err := Validate(c.rule, c.targets)
		if (err == nil) != c.ok {
			t.Errorf("%s: got err=%v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestValidateClearsUnusedFields(t *testing.T) {
	r := &Rule{Type: "partner", Value: 9, RoleID: "758641373074423808"}
	if err := Validate(r, []string{"bot"}); err != nil {
		t.Fatal(err)
	}
	if r.Value != 0 || r.RoleID != "" {
		t.Fatalf("unused fields not cleared: %+v", r)
	}
}

func TestEveryKindHasQueriesForItsTargets(t *testing.T) {
	for _, k := range Kinds {
		if k.Type == "server_booster" || k.Type == "has_role" {
			continue
		}
		for _, target := range k.Targets {
			if _, ok := queries[k.Type][target]; !ok {
				t.Errorf("%s has no query for %s", k.Type, target)
			}
		}
	}
}
