package validators

import (
	"context"
	"testing"

	"popplio/config"
)

// The staging/dev branch calls perms.StaffPerms, which hits the database, so
// it isn't covered here. The prod short-circuit is pure and worth pinning
// down on its own: it's the difference between every request skipping this
// check entirely (if the env comparison is ever inverted) and every prod
// request paying for a permission lookup that can never deny it anything.
func TestStagingCheckSensitiveSkipsTheCheckInProd(t *testing.T) {
	prev := config.CurrentEnv
	defer func() { config.CurrentEnv = prev }()

	config.CurrentEnv = config.CurrentEnvProd

	if err := StagingCheckSensitive(context.Background(), "some-user-id"); err != nil {
		t.Errorf("expected no error in prod, got %v", err)
	}
}
