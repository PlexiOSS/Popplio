package impls

import (
	"github.com/PlexiOSS/Keel/uuidutil"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDString renders a pgtype.UUID the way Rust's Uuid::hyphenated does. An
// invalid (NULL) uuid renders as the empty string.
func UUIDString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}

	return uuidutil.Encode(u.Bytes)
}

// UUIDStrings maps a slice of pgtype.UUIDs to their hyphenated forms.
func UUIDStrings(ids []pgtype.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, UUIDString(id))
	}

	return out
}
