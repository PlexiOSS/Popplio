package bot

import (
	"errors"
	"testing"

	"popplio/config"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

func TestGuardOmiGuild(t *testing.T) {
	prevCfg, prevLog, prevIn, prevLeave, prevReport := state.Config, state.Logger, arcadiaInGuild, leaveGuild, reportLeft
	t.Cleanup(func() {
		state.Config, state.Logger, arcadiaInGuild, leaveGuild, reportLeft = prevCfg, prevLog, prevIn, prevLeave, prevReport
	})

	state.Logger = zap.NewNop()
	state.Config = &config.Config{Servers: config.Servers{Main: 1, Staff: 2, Testing: 3}}

	arcadia := map[snowflake.ID]bool{10: true}
	arcadiaInGuild = func(id snowflake.ID) (bool, error) {
		if id == 99 {
			return false, errors.New("discord is down")
		}

		return arcadia[id], nil
	}

	var left, reported []snowflake.ID
	leaveGuild = func(_ *bot.Client, id snowflake.ID) error {
		left = append(left, id)
		return nil
	}
	reportLeft = func(_ *bot.Client, id snowflake.ID, _ string) { reported = append(reported, id) }

	cases := []struct {
		id    snowflake.ID
		leave bool
	}{
		{1, false},
		{2, false},
		{3, false},
		{10, false},
		{11, true},
		{99, false},
	}

	for _, c := range cases {
		if got := guardOmiGuild(nil, c.id, "x"); got != c.leave {
			t.Errorf("guild %d: left=%v, want %v", c.id, got, c.leave)
		}
	}

	if len(left) != 1 || left[0] != 11 || len(reported) != 1 || reported[0] != 11 {
		t.Fatalf("expected to leave and report only guild 11, left %v reported %v", left, reported)
	}
}

func TestGuardOmiGuildStaysWhenLeavingFails(t *testing.T) {
	prevCfg, prevLog, prevIn, prevLeave, prevReport := state.Config, state.Logger, arcadiaInGuild, leaveGuild, reportLeft
	t.Cleanup(func() {
		state.Config, state.Logger, arcadiaInGuild, leaveGuild, reportLeft = prevCfg, prevLog, prevIn, prevLeave, prevReport
	})

	state.Logger = zap.NewNop()
	state.Config = &config.Config{Servers: config.Servers{Main: 1}}
	arcadiaInGuild = func(snowflake.ID) (bool, error) { return false, nil }
	leaveGuild = func(*bot.Client, snowflake.ID) error { return errors.New("nope") }
	reportLeft = func(*bot.Client, snowflake.ID, string) { t.Fatal("should not report a failed leave") }

	if guardOmiGuild(nil, 5, "x") {
		t.Fatal("a failed leave should not count as leaving")
	}
}
