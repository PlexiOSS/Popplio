// Copyright (C) 2026 NodeByte LTD

package validators

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

const (
	PermissionAdministrator uint64 = 1 << 3
	NoAdminURL                     = "https://noadmin.info"
)

func NoAdminReportURL(invite string) string {
	return NoAdminURL + "/analyze?" + url.Values{"invite": {invite}}.Encode()
}

func IsDiscordAuthorizeURL(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	for _, prefix := range []string{"www.", "canary.", "ptb."} {
		host = strings.TrimPrefix(host, prefix)
	}
	return (host == "discord.com" || host == "discordapp.com") && strings.Contains(u.Path, "oauth2/authorize")
}

func ValidateBotInvite(invite string) error {
	u, err := url.Parse(strings.TrimSpace(invite))

	if err != nil || !IsDiscordAuthorizeURL(u) {
		return nil
	}

	raw, ok := u.Query()["permissions"]

	if !ok || len(raw) == 0 || raw[0] == "" {
		return nil
	}

	value, err := strconv.ParseUint(raw[0], 10, 64)

	if err != nil {
		return errors.New("your invite link has an invalid permissions value. Build a new link at " + NoAdminURL + "/calculator")
	}

	if value&PermissionAdministrator != 0 {
		return errors.New("your invite link requests the Administrator permission, which is not allowed on Omniplex. Request only the permissions your bot actually uses. See " + NoAdminReportURL(invite) + " for what your link asks for, and " + NoAdminURL + "/calculator to build a new one")
	}

	return nil
}
