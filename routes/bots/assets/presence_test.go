// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"testing"
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestApplySelfStatusMarksUnknownPresence(t *testing.T) {
	none := pgtype.Timestamptz{}
	recent := pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true}
	old := pgtype.Timestamptz{Time: time.Now().Add(-8 * 24 * time.Hour), Valid: true}

	shared := map[string]any{"cache": "db"}
	unknown := &dovetypes.PlatformUser{ExtraData: shared}
	ApplySelfStatus(unknown, "", 0, none, "", none)

	if unknown.Status != dovetypes.PlatformStatusOffline || unknown.ExtraData["presence_unknown"] != true {
		t.Fatalf("no presence data = %q %v, want offline and presence_unknown", unknown.Status, unknown.ExtraData)
	}

	if _, ok := shared["presence_unknown"]; ok {
		t.Fatal("the shared extra data map was changed")
	}

	posted := &dovetypes.PlatformUser{ExtraData: map[string]any{}}
	ApplySelfStatus(posted, "", 10, pgtype.Timestamptz{Time: time.Now(), Valid: true}, "", none)

	if posted.Status != dovetypes.PlatformStatusOnline || posted.ExtraData["presence_unknown"] != nil {
		t.Fatalf("recent stats post = %q %v, want online and known", posted.Status, posted.ExtraData)
	}

	self := &dovetypes.PlatformUser{}
	ApplySelfStatus(self, "offline", 0, none, "", none)

	if self.Status != dovetypes.PlatformStatusOffline || self.ExtraData["presence_unknown"] != nil {
		t.Fatalf("posted offline = %q %v, want offline and known", self.Status, self.ExtraData)
	}

	recentlyOnline := &dovetypes.PlatformUser{}
	ApplySelfStatus(recentlyOnline, "", 0, none, "offline", recent)

	if recentlyOnline.Status != dovetypes.PlatformStatusOffline || recentlyOnline.ExtraData["presence_unknown"] != nil {
		t.Fatalf("offline after being seen recently = %q %v, want offline and known", recentlyOnline.Status, recentlyOnline.ExtraData)
	}

	for name, seen := range map[string]pgtype.Timestamptz{"never": none, "over a week ago": old} {
		user := &dovetypes.PlatformUser{}
		ApplySelfStatus(user, "", 0, none, "offline", seen)

		if user.ExtraData["presence_unknown"] != true {
			t.Fatalf("offline, last seen %s = %v, want presence_unknown", name, user.ExtraData)
		}

		platform := &dovetypes.PlatformUser{Status: dovetypes.PlatformStatusOffline, ExtraData: map[string]any{"cache": "platform"}}
		ApplySelfStatus(platform, "", 0, none, "", seen)

		if platform.ExtraData["presence_unknown"] != true {
			t.Fatalf("gateway offline, last seen %s = %v, want presence_unknown", name, platform.ExtraData)
		}
	}

	live := &dovetypes.PlatformUser{Status: dovetypes.PlatformStatusIdle, ExtraData: map[string]any{"cache": "platform"}}
	ApplySelfStatus(live, "", 0, none, "", none)

	if live.Status != dovetypes.PlatformStatusIdle || live.ExtraData["presence_unknown"] != nil {
		t.Fatalf("gateway idle = %q %v", live.Status, live.ExtraData)
	}
}
