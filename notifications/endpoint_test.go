// Copyright (C) 2026 NodeByte LTD

package notifications

import "testing"

func TestValidPushEndpoint(t *testing.T) {
	cases := map[string]bool{
		"https://fcm.googleapis.com/fcm/send/abc":                true,
		"https://updates.push.services.mozilla.com/wpush/v2/abc": true,
		"https://web.push.apple.com/QGx":                         true,
		"https://wns2-by3p.notify.windows.com/w/?token=abc":      true,
		"http://fcm.googleapis.com/fcm/send/abc":                 false,
		"https://fcm.googleapis.com:8443/fcm/send/abc":           false,
		"https://evilfcm.googleapis.com.attacker.net/x":          false,
		"https://notify.windows.com.attacker.net/x":              false,
		"https://user@fcm.googleapis.com/x":                      false,
		"https://127.0.0.1/x":                                    false,
		"http://169.254.169.254/latest/meta-data":                false,
		"https://attackerpush.apple.com/x":                       false,
		"":                                                       false,
		"not a url":                                              false,
	}

	for raw, want := range cases {
		if got := ValidPushEndpoint(raw); got != want {
			t.Errorf("ValidPushEndpoint(%q) = %v, want %v", raw, got, want)
		}
	}
}
