package get_audit_events

import (
	"net/http/httptest"
	"testing"

	"popplio/testdb"
	"popplio/types"
)

func TestResolveNames(t *testing.T) {
	testdb.Pool(t)

	userID, botID, unknown := testdb.ID(t), testdb.ID(t), testdb.ID(t)

	testdb.Exec(t, `INSERT INTO internal_user_cache__discord (id, username, display_name, avatar, bot) VALUES
		($1, 'alice', 'Alice', 'https://cdn.example/alice.png', false),
		($2, 'somebot', '', '', true)`, userID, botID)
	testdb.Cleanup(t, "DELETE FROM internal_user_cache__discord WHERE id = ANY($1)", []string{userID, botID})

	events := []types.AuditEvent{
		{ActorType: "user", ActorID: &userID, TargetID: &botID},
		{ActorType: "user", ActorID: &unknown, TargetID: &userID},
		{ActorType: "system"},
	}

	resolveNames(httptest.NewRequest("GET", "/", nil), events)

	if n := events[0].ActorName; n == nil || *n != "Alice" {
		t.Errorf("actor name = %v, want Alice", n)
	}

	if a := events[0].ActorAvatar; a == nil || *a != "https://cdn.example/alice.png" {
		t.Errorf("actor avatar = %v", a)
	}

	if n := events[0].TargetName; n == nil || *n != "somebot" {
		t.Errorf("target name = %v, want somebot (username when there is no display name)", n)
	}

	if events[1].ActorName != nil {
		t.Errorf("unknown actor got a name: %v", *events[1].ActorName)
	}

	if n := events[1].TargetName; n == nil || *n != "Alice" {
		t.Errorf("target name = %v, want Alice", n)
	}
}
