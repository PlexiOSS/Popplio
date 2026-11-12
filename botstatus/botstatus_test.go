// Copyright (C) 2026 NodeByte LTD

package botstatus

import "testing"

func init() {
	siteHost = func() string { return "omniplex.gg" }
}

func TestNumber(t *testing.T) {
	cases := map[int64]string{0: "0", 7: "7", 999: "999", 1000: "1,000", 12345: "12,345", 1234567: "1,234,567"}

	for n, want := range cases {
		if got := Number(n); got != want {
			t.Errorf("Number(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestArcadiaQueueLine(t *testing.T) {
	if got := Arcadia(Counts{})[0].Text; got != "an empty review queue" {
		t.Errorf("empty queue: %q", got)
	}

	if got := Arcadia(Counts{PendingBots: 1})[0].Text; got != "Reviewing 1 listing in the queue" {
		t.Errorf("one waiting: %q", got)
	}

	if got := Arcadia(Counts{PendingBots: 2, PendingServers: 1})[0].Text; got != "Reviewing 3 listings in the queue" {
		t.Errorf("three waiting: %q", got)
	}
}

func TestServersLines(t *testing.T) {
	if got := Servers(Counts{ListedServers: 12})[0].Text; got != "12 listed servers" {
		t.Errorf("servers line: %q", got)
	}
}

func TestOmiLines(t *testing.T) {
	lines := Omi(Counts{ListedBots: 910, ListedServers: 12})

	if lines[0].Text != "910 bots and 12 servers" || lines[2].Text != "Find your next bot at omniplex.gg" {
		t.Errorf("unexpected Omi lines: %+v", lines)
	}
}
