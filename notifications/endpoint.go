// Copyright (C) 2026 NodeByte LTD

package notifications

import (
	"net/url"
	"strings"
	"time"

	"popplio/netguard"
)

var pushHosts = []string{
	"fcm.googleapis.com",
	"android.googleapis.com",
	"updates.push.services.mozilla.com",
	"push.apple.com",
	"notify.windows.com",
}

var pushClient = netguard.NewClient(15 * time.Second)

func ValidPushEndpoint(raw string) bool {
	u, err := url.Parse(raw)

	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}

	host := strings.ToLower(u.Hostname())

	for _, h := range pushHosts {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}

	return false
}
