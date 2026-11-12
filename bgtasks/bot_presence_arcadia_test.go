// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
)

func TestPresenceStatus(t *testing.T) {
	cases := []struct {
		presence discord.Presence
		ok       bool
		want     string
	}{
		{discord.Presence{}, false, "offline"},
		{discord.Presence{Status: discord.OnlineStatusOnline}, true, "online"},
		{discord.Presence{Status: discord.OnlineStatusIdle}, true, "idle"},
		{discord.Presence{Status: discord.OnlineStatusDND}, true, "dnd"},
		{discord.Presence{Status: discord.OnlineStatusInvisible}, true, "offline"},
		{discord.Presence{Status: discord.OnlineStatusOffline}, true, "offline"},
	}

	for _, c := range cases {
		if got := presenceStatus(c.presence, c.ok); got != c.want {
			t.Errorf("presenceStatus(%q, %v) = %q, want %q", c.presence.Status, c.ok, got, c.want)
		}
	}
}

func TestArcadiaPresencesWithoutArcadia(t *testing.T) {
	if got := arcadiaPresences([]string{"1"}); len(got) != 0 {
		t.Errorf("expected nothing without an Arcadia connection, got %v", got)
	}
}
