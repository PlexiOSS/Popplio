package bot

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"popplio/perms"

	"github.com/disgoorg/disgo/discord"
)

// The menus have to fit inside Discord's caps, and the catalogue is free to grow
// past them: a category with 26 permissions would otherwise be rejected outright
// at render time, in production, with no test having noticed.
func TestPermissionSelectFitsDiscordLimits(t *testing.T) {
	manager := managerContext{perms: perms.Staff.NewSet(perms.StaffAdministrator)}

	for _, category := range perms.Staff.Categories() {
		s := &permSession{Category: category}

		menu, ok := permissionSelect(s, map[perms.Perm]bool{}, manager)

		if !ok {
			t.Fatalf("category %q rendered no menu", category)
		}

		if len(menu.Options) > discordSelectLimit {
			t.Errorf("category %q has %d options, over Discord's cap", category, len(menu.Options))
		}

		if len(menu.Placeholder) > 100 {
			t.Errorf("category %q has a %d character placeholder", category, len(menu.Placeholder))
		}

		for _, opt := range menu.Options {
			if len(opt.Label) > 100 {
				t.Errorf("%q has a %d character label", opt.Value, len(opt.Label))
			}

			if len(opt.Description) > 100 {
				t.Errorf("%q has a %d character description", opt.Value, len(opt.Description))
			}
		}
	}

	if len(perms.Staff.Categories()) > discordSelectLimit {
		t.Errorf("%d categories will not fit in one select menu", len(perms.Staff.Categories()))
	}
}

// What is already held must come back preselected, or submitting the menu
// unchanged would read as "revoke everything in this category".
func TestPermissionSelectPreselectsHeld(t *testing.T) {
	def, _ := perms.Staff.Lookup(perms.StaffReviewEntities)
	s := &permSession{Category: def.Category}

	held := map[perms.Perm]bool{perms.StaffReviewEntities: true}

	menu, ok := permissionSelect(s, held, managerContext{perms: perms.Staff.NewSet(perms.StaffAdministrator)})

	if !ok {
		t.Fatal("the review category rendered no menu")
	}

	var found bool

	for _, opt := range menu.Options {
		if opt.Value != string(perms.StaffReviewEntities) {
			continue
		}

		found = true

		if !opt.Default {
			t.Error("a held permission should be preselected")
		}
	}

	if !found {
		t.Fatalf("review_entities is missing from the %q menu", def.Category)
	}

	// Nothing may be selected, and everything may be, so a category can be
	// emptied or filled in one submission.
	if menu.MinValues == nil || *menu.MinValues != 0 {
		t.Error("the menu must allow an empty selection, which is how a category is cleared")
	}

	if menu.MaxValues != len(menu.Options) {
		t.Errorf("the menu allows %d of %d options to be selected", menu.MaxValues, len(menu.Options))
	}
}

// A permission the caller cannot manage is shown, marked, rather than hidden:
// they still need to see what a role holds.
func TestPermissionSelectMarksUnmanageable(t *testing.T) {
	def, _ := perms.Staff.Lookup(perms.StaffForceRemoveEntities)

	// A manager who can review bots but not force remove them.
	manager := managerContext{perms: perms.Staff.NewSet(perms.StaffReviewEntities)}

	menu, ok := permissionSelect(&permSession{Category: def.Category}, map[perms.Perm]bool{}, manager)

	if !ok {
		t.Fatal("the category rendered no menu")
	}

	for _, opt := range menu.Options {
		switch opt.Value {
		case string(perms.StaffForceRemoveEntities):
			if !strings.Contains(opt.Label, "🔒") {
				t.Errorf("an unmanageable permission should be marked, got %q", opt.Label)
			}

			// It is dangerous as well, and both marks belong on it.
			if !strings.Contains(opt.Label, "⚠️") {
				t.Errorf("a dangerous permission should be marked, got %q", opt.Label)
			}
		case string(perms.StaffReviewEntities):
			if strings.Contains(opt.Label, "🔒") {
				t.Errorf("a manageable permission should not be marked, got %q", opt.Label)
			}
		}
	}
}

// The role picker offers only what the caller may actually edit, so a click can
// never be refused for a reason the menu already knew about.
func TestRoleSelectRowHidesSeniorRoles(t *testing.T) {
	roles := []staffRole{
		{ID: "a", Name: "Owner", Index: 1},
		{ID: "b", Name: "Head Admin", Index: 3},
		{ID: "c", Name: "Moderator", Index: 5},
	}

	row, ok := roleSelectRow(roles, &permSession{RoleID: "c"}, managerContext{rank: 3})

	if !ok {
		t.Fatal("a manager with a junior role below them should get a picker")
	}

	menu, valid := row.(discord.ActionRowComponent)[0].(discord.StringSelectMenuComponent)

	if !valid {
		t.Fatal("the role picker is not a string select menu")
	}

	if len(menu.Options) != 1 || menu.Options[0].Value != "c" {
		t.Fatalf("only the junior role should be offered, got %+v", menu.Options)
	}

	if !menu.Options[0].Default {
		t.Error("the role being edited should be preselected")
	}

	// A manager at the bottom of the hierarchy has nothing to pick.
	if _, ok := roleSelectRow(roles, &permSession{}, managerContext{rank: perms.NoRank}); ok {
		t.Error("a manager who outranks nobody should get no picker")
	}
}

// "Grant all" and "Revoke all" must never build a selection that the write path
// will refuse: they leave permissions the caller cannot manage exactly as they
// are, so one locked permission does not make the button useless.
func TestBulkCategorySelectionLeavesUnmanageableAlone(t *testing.T) {
	def, _ := perms.Staff.Lookup(perms.StaffTransferBots)
	category := def.Category

	// Can transfer, cannot force remove.
	manager := managerContext{perms: perms.Staff.NewSet(perms.StaffTransferBots)}

	held := map[perms.Perm]bool{perms.StaffForceRemoveEntities: true}

	granted := bulkCategorySelection(category, held, manager, true)

	if !slices.Contains(granted, string(perms.StaffTransferBots)) {
		t.Error("grant all should turn on a permission the caller manages")
	}

	if !slices.Contains(granted, string(perms.StaffForceRemoveEntities)) {
		t.Error("grant all must keep an unmanageable permission that is already held, or the write is refused")
	}

	revoked := bulkCategorySelection(category, held, manager, false)

	if slices.Contains(revoked, string(perms.StaffTransferBots)) {
		t.Error("revoke all should turn off a permission the caller manages")
	}

	if !slices.Contains(revoked, string(perms.StaffForceRemoveEntities)) {
		t.Error("revoke all must leave an unmanageable permission held, not try to take it")
	}

	// A caller who manages nothing in the category changes nothing with either
	// button, rather than submitting a change that will bounce.
	none := managerContext{perms: perms.Staff.NewSet()}

	if got := bulkCategorySelection(category, held, none, true); len(got) != 1 || got[0] != string(perms.StaffForceRemoveEntities) {
		t.Errorf("grant all with no authority should be a no-op, got %v", got)
	}
}

// The buttons are the other half of "interactive", and which ones are offered
// depends on where the editor is.
func TestEditorButtons(t *testing.T) {
	ids := func(components []discord.InteractiveComponent) []string {
		out := make([]string, 0, len(components))

		for _, c := range components {
			out = append(out, c.ID())
		}

		return out
	}

	// On the role picker: nothing to act on yet.
	got := ids(editorButtons(&permSession{}, false))

	if len(got) != 1 || got[0] != permButtonClose {
		t.Errorf("the picker should offer only Close, got %v", got)
	}

	// A role open, no category yet: no bulk actions, but a way back.
	got = ids(editorButtons(&permSession{RoleID: "abc"}, true))

	if slices.Contains(got, permButtonGrantAll) {
		t.Errorf("grant all needs a category open, got %v", got)
	}

	if !slices.Contains(got, permButtonBack) {
		t.Errorf("a role being edited should offer a way back to the picker, got %v", got)
	}

	// A category open: everything.
	got = ids(editorButtons(&permSession{RoleID: "abc", Category: "Bot Reviews"}, true))

	for _, want := range []string{permButtonGrantAll, permButtonClearAll, permButtonBack, permButtonClose} {
		if !slices.Contains(got, want) {
			t.Errorf("%q is missing from %v", want, got)
		}
	}

	// Discord caps an action row at five components.
	if len(got) > 5 {
		t.Errorf("%d buttons will not fit in one action row", len(got))
	}

	// A member editor has no role picker to go back to.
	got = ids(editorButtons(&permSession{UserID: "42", Category: "Bot Reviews"}, true))

	if slices.Contains(got, permButtonBack) {
		t.Errorf("a member editor should not offer the role picker, got %v", got)
	}
}

// The category counts are what tell someone where to look before they open
// anything.
func TestCategorySelectCountsHeld(t *testing.T) {
	def, _ := perms.Staff.Lookup(perms.StaffReviewEntities)

	menu := categorySelect(&permSession{Category: def.Category}, map[perms.Perm]bool{perms.StaffReviewEntities: true})

	var found bool

	for _, opt := range menu.Options {
		if opt.Value != def.Category {
			continue
		}

		found = true

		want := fmt.Sprintf("1 of %d granted", len(perms.Staff.InCategory(def.Category)))

		if opt.Description != want {
			t.Errorf("description = %q, want %q", opt.Description, want)
		}

		if !opt.Default {
			t.Error("the open category should be preselected")
		}
	}

	if !found {
		t.Fatalf("category %q is missing from the picker", def.Category)
	}
}

// heldMap reads the stored list exactly. Administrator implies everything for an
// access check, but an editor that ticked every box for an administrator role
// would write those permissions out on the next submission.
func TestHeldMapIsExact(t *testing.T) {
	held := heldMap([]perms.Perm{perms.StaffAdministrator})

	if !held[perms.StaffAdministrator] {
		t.Error("administrator should be held")
	}

	if held[perms.StaffReviewEntities] {
		t.Error("administrator implies review_entities but does not store it, so the box must be unticked")
	}
}

func TestPermChangeSummary(t *testing.T) {
	role := permTarget{role: staffRole{Name: "Moderator", RoleID: "123"}, holders: 4}

	got := permChangeSummary(&permSession{RoleID: "x"}, role,
		[]perms.Perm{perms.StaffReviewEntities}, []perms.Perm{perms.StaffCertifyEntities})

	for _, want := range []string{"granted ``review_entities``", "revoked ``certify_entities``", "4 member(s)"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary %q should contain %q", got, want)
		}
	}

	// A member's change reaches only them, so it must not claim a member count.
	got = permChangeSummary(&permSession{UserID: "42"}, permTarget{holders: 1}, []perms.Perm{perms.StaffViewPanel}, nil)

	if !strings.Contains(got, "<@42>") || strings.Contains(got, "member(s) with") {
		t.Errorf("a member summary should name the member and nothing else, got %q", got)
	}
}
