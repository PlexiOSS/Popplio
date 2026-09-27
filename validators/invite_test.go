// Copyright (C) 2026 NodeByte LTD

package validators

import (
	"strings"
	"testing"
)

func TestValidateBotInvite(t *testing.T) {
	cases := []struct {
		invite  string
		blocked bool
	}{
		{"https://discord.com/oauth2/authorize?client_id=1&permissions=8&scope=bot", true},
		{"https://discord.com/api/oauth2/authorize?client_id=1&permissions=2147483656&scope=bot+applications.commands", true},
		{"https://canary.discord.com/oauth2/authorize?client_id=1&permissions=8&scope=bot", true},
		{"https://discordapp.com/oauth2/authorize?client_id=1&permissions=8&scope=bot", true},
		{"https://www.discord.com/oauth2/authorize?client_id=1&permissions=9&scope=bot", true},
		{"https://discord.com/oauth2/authorize?client_id=1&permissions=abc&scope=bot", true},
		{"https://discord.com/oauth2/authorize?client_id=1&permissions=99999999999999999999999&scope=bot", true},
		{"https://discord.com/oauth2/authorize?client_id=1&permissions=3165184&scope=bot", false},
		{"https://discord.com/oauth2/authorize?client_id=1&permissions=0&scope=bot", false},
		{"https://discord.com/oauth2/authorize?client_id=1&scope=applications.commands", false},
		{"https://discord.com/oauth2/authorize?client_id=1&permissions=&scope=bot", false},
		{"https://mybot.gg/invite?permissions=8", false},
		{"https://discord.com/application-directory/1", false},
	}

	for _, c := range cases {
		err := ValidateBotInvite(c.invite)
		if (err != nil) != c.blocked {
			t.Errorf("%s: blocked=%v, err=%v", c.invite, c.blocked, err)
		}
	}
}

func TestValidateBotInviteMessageLinksReport(t *testing.T) {
	invite := "https://discord.com/oauth2/authorize?client_id=1&permissions=8&scope=bot"
	err := ValidateBotInvite(invite)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), NoAdminReportURL(invite)) {
		t.Fatalf("message does not link the report: %s", err)
	}
	if strings.Contains(err.Error(), "—") {
		t.Fatal("message contains an em dash")
	}
}
