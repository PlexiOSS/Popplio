// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"fmt"
	"slices"
	"testing"
)

func TestPickDueBotsGivesEveryBotATurn(t *testing.T) {
	candidates := make([]string, 50)

	for i := range candidates {
		candidates[i] = fmt.Sprint(i)
	}

	tracked := map[string]bool{"0": true, "1": true}
	seen := map[string]bool{}

	for range 200 {
		due, untracked := pickDueBots(candidates, tracked, 5)

		if len(due) != 5 || untracked != 48 {
			t.Fatalf("due = %d, untracked = %d, want 5 and 48", len(due), untracked)
		}

		if slices.Contains(due, "0") || slices.Contains(due, "1") {
			t.Fatalf("picked a tracked bot: %v", due)
		}

		for _, id := range due {
			seen[id] = true
		}
	}

	if len(seen) != 48 {
		t.Fatalf("%d of 48 bots got a turn", len(seen))
	}

	if due, untracked := pickDueBots(candidates[:3], nil, 5); len(due) != 3 || untracked != 3 {
		t.Fatalf("short list: due = %d, untracked = %d", len(due), untracked)
	}
}
