// Copyright (C) 2026 NodeByte LTD

package apps

import (
	"strings"
	"testing"

	"popplio/types"
)

func TestCheckAnswer(t *testing.T) {
	choice := types.Question{Type: KindChoice, Options: []string{"Mornings", "Evenings"}}
	multi := types.Question{Type: KindMulti, Options: []string{"Go", "Rust", "TypeScript"}}

	cases := []struct {
		name string
		q    types.Question
		ans  string
		ok   bool
	}{
		{"legacy short", types.Question{Short: true}, "hi", true},
		{"legacy long too short", types.Question{}, "too short", false},
		{"legacy long", types.Question{}, strings.Repeat("a", 60), true},
		{"custom min", types.Question{Type: KindLong, MinLength: 10}, "ten chars!", true},
		{"custom max", types.Question{Type: KindShort, MaxLength: 5}, "toolong", false},
		{"choice", choice, "Evenings", true},
		{"choice not offered", choice, "Nights", false},
		{"multi", multi, "Go\nRust", true},
		{"multi repeated", multi, "Go\nGo", false},
		{"multi unknown", multi, "Go\nJava", false},
		{"yes", types.Question{Type: KindYesNo}, "Yes", true},
		{"maybe", types.Question{Type: KindYesNo}, "Maybe", false},
		{"number", types.Question{Type: KindNumber}, "1250", true},
		{"not a number", types.Question{Type: KindNumber}, "lots", false},
		{"url", types.Question{Type: KindURL}, "https://github.com/omniplex", true},
		{"bad url", types.Question{Type: KindURL}, "github.com", false},
		{"rating", types.Question{Type: KindRating}, "4", true},
		{"rating out of range", types.Question{Type: KindRating}, "6", false},
	}

	for _, c := range cases {
		if err := CheckAnswer(c.q, c.ans); (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestVisible(t *testing.T) {
	q := types.Question{ShowIf: &types.QuestionCondition{QuestionID: "langs", Values: []string{"Go"}}}

	if Visible(q, map[string]string{"langs": "Rust"}) {
		t.Fatal("shown without the right answer")
	}

	if !Visible(q, map[string]string{"langs": "Rust\nGo"}) {
		t.Fatal("hidden although Go was picked")
	}

	if !Visible(types.Question{}, nil) {
		t.Fatal("a question with no condition was hidden")
	}
}

func TestCheckDefinition(t *testing.T) {
	valid := []types.Question{
		{ID: "role", Type: KindChoice, Options: []string{"Reviewer", "Moderator"}},
		{ID: "why", Type: KindLong, ShowIf: &types.QuestionCondition{QuestionID: "role", Values: []string{"Reviewer"}}},
	}

	if msg := CheckDefinition(valid); msg != "" {
		t.Fatalf("valid form rejected: %s", msg)
	}

	bad := map[string][]types.Question{
		"one option":        {{ID: "a", Type: KindChoice, Options: []string{"Only"}}},
		"duplicate options": {{ID: "a", Type: KindMulti, Options: []string{"X", "X"}}},
		"options on text":   {{ID: "a", Type: KindShort, Options: []string{"X", "Y"}}},
		"unknown type":      {{ID: "a", Type: "date"}},
		"min over max":      {{ID: "a", Type: KindLong, MinLength: 100, MaxLength: 50}},
		"depends on later":  {{ID: "a", ShowIf: &types.QuestionCondition{QuestionID: "b", Values: []string{"Yes"}}}, {ID: "b", Type: KindYesNo}},
		"depends on text":   {{ID: "a", Type: KindShort}, {ID: "b", ShowIf: &types.QuestionCondition{QuestionID: "a", Values: []string{"x"}}}},
		"value not offered": {{ID: "a", Type: KindYesNo}, {ID: "b", ShowIf: &types.QuestionCondition{QuestionID: "a", Values: []string{"Maybe"}}}},
		"max too high":      {{ID: "a", Type: KindShort, MaxLength: 9000}},
		"flag on text":      {{ID: "a", Type: KindShort, Flag: []string{"x"}}},
		"flag not offered":  {{ID: "a", Type: KindYesNo, Flag: []string{"Maybe"}}},
	}

	for name, qs := range bad {
		if CheckDefinition(qs) == "" {
			t.Errorf("%s: accepted", name)
		}
	}
}
