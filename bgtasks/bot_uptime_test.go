// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestDecideUptime(t *testing.T) {
	recent := pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true}
	old := pgtype.Timestamptz{Time: time.Now().Add(-8 * 24 * time.Hour), Valid: true}

	cases := []struct {
		name          string
		signals       uptimeSignals
		online, known bool
	}{
		{"nothing known", uptimeSignals{}, false, false},
		{"posted online", uptimeSignals{selfStatus: "online", statsFresh: true}, true, true},
		{"posted offline is always trusted", uptimeSignals{selfStatus: "offline", statsFresh: true}, false, true},
		{"old posted status is ignored", uptimeSignals{selfStatus: "offline"}, false, false},
		{"presence online", uptimeSignals{presence: "dnd"}, true, true},
		{"offline but never seen online, like an HTTP bot", uptimeSignals{presence: "offline"}, false, false},
		{"offline and seen online recently", uptimeSignals{presence: "offline", lastSeenOnline: recent}, false, true},
		{"offline and not seen online for over a week", uptimeSignals{presence: "offline", lastSeenOnline: old}, false, false},
		{"recent stats outweigh an offline presence", uptimeSignals{presence: "offline", statsFresh: true, servers: 5, lastSeenOnline: recent}, true, true},
	}

	for _, c := range cases {
		online, known := decideUptime(c.signals)

		if online != c.online || known != c.known {
			t.Errorf("%s: got online=%v known=%v, want online=%v known=%v", c.name, online, known, c.online, c.known)
		}
	}
}
