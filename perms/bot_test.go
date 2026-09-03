// Copyright (C) 2026 NodeByte LTD

package perms

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func TestBotAccountHoldsNothing(t *testing.T) {
	roles := []Role{{ID: "admins", Index: 1, Perms: []Perm{StaffAdministrator}}}

	cases := []struct {
		name   string
		grants StaffGrants
	}{
		{"roles", StaffGrants{Roles: roles, BotAccount: true}},
		{"direct grants", StaffGrants{Extras: []Perm{StaffReviewEntities, StaffViewPanel}, BotAccount: true}},
		{"the owners list", StaffGrants{ConfigOwner: true, BotAccount: true}},
		{"everything at once", StaffGrants{
			Roles:       roles,
			Extras:      []Perm{StaffReviewEntities},
			ConfigOwner: true,
			BotAccount:  true,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := tc.grants.Resolve()

			if !resolved.IsEmpty() {
				t.Errorf("a bot resolved to %v", resolved.Strings())
			}

			if resolved.IsSuper() {
				t.Error("a bot must never read as super")
			}

			for _, p := range []Perm{StaffAdministrator, StaffReviewEntities, StaffViewPanel, StaffViewStaff} {
				if resolved.Has(p) {
					t.Errorf("a bot should not hold %s", p)
				}
			}
		})
	}
}

func TestBotAccountHasNoRank(t *testing.T) {
	withOwners(t, snowflake.ID(510065483693817867))

	bot := StaffGrants{
		Roles:       []Role{{ID: "admins", Index: 1, Perms: []Perm{StaffAdministrator}}},
		ConfigOwner: true,
		BotAccount:  true,
	}

	if got := bot.Rank(); got != NoRank {
		t.Errorf("Rank() = %d, want NoRank (%d)", got, NoRank)
	}

	person := bot
	person.BotAccount = false

	if got := person.Rank(); got != OwnerRank {
		t.Errorf("a person with these grants should rank %d, got %d", OwnerRank, got)
	}
}

func TestBotAccountCannotBePatchedInto(t *testing.T) {
	bot := StaffGrants{BotAccount: true, Extras: []Perm{StaffReviewEntities}}

	current := bot.Resolve()
	next := current.With(StaffReviewEntities)

	manager := Staff.NewSet(StaffAdministrator)

	if err := CheckPatch(manager, current, next); err != nil {
		t.Fatalf("the patch should be legitimate for the manager: %v", err)
	}

	if !bot.Resolve().IsEmpty() {
		t.Error("writing a grant onto a bot must not resolve into anything")
	}
}
