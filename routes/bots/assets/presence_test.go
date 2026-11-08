// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"testing"
	"time"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestApplySelfStatusMarksUnknownPresence(t *testing.T) {
	shared := map[string]any{"cache": "db"}
	unknown := &dovetypes.PlatformUser{ExtraData: shared}
	ApplySelfStatus(unknown, "", 0, pgtype.Timestamptz{}, "")

	if unknown.Status != dovetypes.PlatformStatusOffline || unknown.ExtraData["presence_unknown"] != true {
		t.Fatalf("no presence data = %q %v, want offline and presence_unknown", unknown.Status, unknown.ExtraData)
	}

	if _, ok := shared["presence_unknown"]; ok {
		t.Fatal("the shared extra data map was changed")
	}

	posted := &dovetypes.PlatformUser{ExtraData: map[string]any{}}
	ApplySelfStatus(posted, "", 10, pgtype.Timestamptz{Time: time.Now(), Valid: true}, "")

	if posted.Status != dovetypes.PlatformStatusOnline || posted.ExtraData["presence_unknown"] != nil {
		t.Fatalf("recent stats post = %q %v, want online and known", posted.Status, posted.ExtraData)
	}

	seen := &dovetypes.PlatformUser{}
	ApplySelfStatus(seen, "", 0, pgtype.Timestamptz{}, "offline")

	if seen.Status != dovetypes.PlatformStatusOffline || seen.ExtraData["presence_unknown"] != nil {
		t.Fatalf("seen offline = %q %v, want offline and known", seen.Status, seen.ExtraData)
	}
}
