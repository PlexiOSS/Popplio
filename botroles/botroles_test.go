package botroles

import (
	"slices"
	"testing"

	"popplio/config"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
)

func withRoles(t *testing.T, partner snowflake.ID) {
	prev := state.Config
	state.Config = &config.Config{Roles: config.Roles{BotRole: 1, CertBot: 2, PartnerBot: partner}}
	t.Cleanup(func() { state.Config = prev })
}

func TestWanted(t *testing.T) {
	withRoles(t, 3)

	cases := []struct {
		name    string
		listing Listing
		want    []snowflake.ID
	}{
		{"not listed", Listing{}, []snowflake.ID{1}},
		{"approved", Listing{Type: "approved"}, []snowflake.ID{1}},
		{"certified", Listing{Type: "certified"}, []snowflake.ID{1, 2}},
		{"partner", Listing{Type: "approved", Partner: true}, []snowflake.ID{1, 3}},
		{"unlisted partner", Listing{Partner: true}, []snowflake.ID{1, 3}},
		{"certified partner", Listing{Type: "certified", Partner: true}, []snowflake.ID{1, 2, 3}},
		{"denied", Listing{Type: "denied"}, []snowflake.ID{1}},
	}

	for _, c := range cases {
		if got := Wanted(c.listing); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWantedSkipsUnsetPartnerRole(t *testing.T) {
	withRoles(t, 0)

	if got := Wanted(Listing{Type: "certified", Partner: true}); !slices.Equal(got, []snowflake.ID{1, 2}) {
		t.Fatalf("got %v", got)
	}
}

func TestMissing(t *testing.T) {
	if got := Missing([]snowflake.ID{1, 9}, []snowflake.ID{1, 2, 0, 3}); !slices.Equal(got, []snowflake.ID{2, 3}) {
		t.Fatalf("got %v", got)
	}

	if got := Missing([]snowflake.ID{1, 2}, []snowflake.ID{1, 2}); got != nil {
		t.Fatalf("nothing should be missing, got %v", got)
	}
}
