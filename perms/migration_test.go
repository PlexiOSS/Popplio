// Copyright (C) 2026 NodeByte LTD

package perms

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const migrationPath = "../db/migrations/20260801004000_flatperms.sql"

var (
	tupleRe    = regexp.MustCompile(`\('(staff|entity)',\s*'([^']+)',\s*'([^']+)'\)`)
	wildcardRe = regexp.MustCompile(`(?s)SELECT '(staff|entity)', '([^']+)', p\s*FROM unnest\(ARRAY\[(.*?)\]\)`)
	quotedRe   = regexp.MustCompile(`'([^']+)'`)
)

func readMigration(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(migrationPath)

	if err != nil {
		t.Fatalf("reading %s: %v", migrationPath, err)
	}

	return string(b)
}

func retired(sql, domain, perm string) bool {
	body := sql

	if _, after, ok := strings.Cut(sql, "INSERT INTO retired_perm"); ok {
		body, _, _ = strings.Cut(after, ";")
	} else {
		return false
	}

	return strings.Contains(body, "('"+domain+"', '"+perm+"')")
}

func catalogueFor(domain string) *Catalogue {
	if domain == "staff" {
		return Staff
	}

	return Entity
}

func TestMigrationTargetsAreDeclared(t *testing.T) {
	sql := readMigration(t)

	for _, m := range tupleRe.FindAllStringSubmatch(sql, -1) {
		domain, old, next := m[1], m[2], m[3]

		if _, ok := catalogueFor(domain).Lookup(Perm(next)); !ok {
			t.Errorf("migration maps %s permission %q to %q, which no catalogue declares", domain, old, next)
		}
	}

	for _, m := range wildcardRe.FindAllStringSubmatch(sql, -1) {
		domain, old, body := m[1], m[2], m[3]

		for _, q := range quotedRe.FindAllStringSubmatch(body, -1) {
			if _, ok := catalogueFor(domain).Lookup(Perm(q[1])); !ok {
				t.Errorf("migration expands %s wildcard %q to %q, which no catalogue declares", domain, old, q[1])
			}
		}
	}
}

func TestEveryLegacyPermissionIsMigrated(t *testing.T) {
	sql := readMigration(t)

	mapped := map[string]map[string]string{
		"staff":  {},
		"entity": {},
	}

	for _, m := range tupleRe.FindAllStringSubmatch(sql, -1) {
		mapped[m[1]][m[2]] = m[3]
	}

	for domain, cat := range map[string]*Catalogue{"staff": Staff, "entity": Entity} {
		for _, d := range cat.Definitions() {
			for _, old := range d.Legacy {
				got, ok := mapped[domain][old]

				if !ok {
					t.Errorf("%s: %q says it replaces %q, but the migration does not map it", domain, d.ID, old)
					continue
				}

				if got != string(d.ID) {
					t.Errorf("%s: %q says it replaces %q, but the migration maps it to %q", domain, d.ID, old, got)
				}
			}
		}
	}
}

func TestMigrationCoversTheOldVocabulary(t *testing.T) {
	sql := readMigration(t)

	handled := func(domain, perm string) bool {
		perm = strings.TrimPrefix(perm, "~")

		for _, m := range tupleRe.FindAllStringSubmatch(sql, -1) {
			if m[1] == domain && m[2] == perm {
				return true
			}
		}

		for _, m := range wildcardRe.FindAllStringSubmatch(sql, -1) {
			if m[1] == domain && m[2] == perm {
				return true
			}
		}

		if retired(sql, domain, perm) {
			return true
		}

		return strings.Contains(sql, "'"+perm+"'")
	}

	staffVocab := []string{
		"global.*", "global.view", "global.view_sensitive", "arcadia.*",
		"rpc.*", "rpc.Claim", "rpc.Unclaim", "rpc.Approve", "rpc.Deny", "rpc.Unverify",
		"~rpc.PremiumAdd", "~rpc.PremiumRemove", "~rpc.VoteBanAdd", "~rpc.VoteBanRemove",
		"rpc_logs.*", "rpc_logs.view", "apps.*", "apps.view",
		"staff_members.*", "staff_members.view", "staff_positions.*", "staff_positions.view",
		"staff_disciplinary_types.*", "persepolis.*", "persepolis.view_onboarding_responses",
		"shop.view", "shop_items.*", "shop_items.view", "shop_item_benefits.*",
		"shop_item_benefits.update", "shop_item_benefits.view", "vote_credit_tiers.*",
		"bot_whitelist.*", "partners.*", "popplio.*", "popplio_staging.*", "borealis.*",
		"cdn.list_scopes", "cdn.list", "cdn.add_file", "cdn.upload_chunk",
		"cdn#ibl@main.*", "cdn#ibl@main.list", "cdn#ibl@main.read_file",
		"developer.marker", "lead_developer.marker", "human_resources.marker",
		"bot_reviewer.marker", "service_account.marker", "dt.marker",
	}

	for _, perm := range staffVocab {
		if !handled("staff", perm) {
			t.Errorf("staff permission %q was in use but the migration does not handle it", perm)
		}
	}

	entityVocab := []string{
		"global.*", "global.add", "global.edit", "global.delete", "global.resubmit",
		"global.set_vanity", "global.request_cert", "global.get_webhooks", "global.edit_webhooks",
		"global.test_webhooks", "global.get_webhook_logs", "global.delete_webhook_logs",
		"global.upload_assets", "global.delete_assets", "global.create_owner_review",
		"global.edit_owner_review", "global.delete_owner_review", "global.redeem_vote_credits",
		"bot.*", "bot.add", "bot.edit", "bot.delete", "bot.resubmit", "bot.request_cert",
		"bot.set_vanity", "bot.get_webhooks", "bot.edit_webhooks", "bot.test_webhooks",
		"bot.get_webhook_logs", "bot.delete_webhook_logs", "bot.upload_assets", "bot.delete_assets",
		"bot.create_owner_review", "bot.edit_owner_review", "bot.delete_owner_review",
		"bot.redeem_vote_credits", "bot.view_api_tokens", "bot.reset_api_tokens",
		"server.*", "team.*", "team.edit", "team.edit_webhooks",
		"team_member.*", "team_member.add", "team_member.edit", "team_member.delete", "team_member.remove",
	}

	for _, perm := range entityVocab {
		if !handled("entity", perm) {
			t.Errorf("entity permission %q was in use but the migration does not handle it", perm)
		}
	}
}

func TestBorealisIsRetired(t *testing.T) {
	for _, d := range Staff.Definitions() {
		if strings.Contains(strings.ToLower(string(d.ID)), "boreal") {
			t.Errorf("the staff catalogue still declares %q", d.ID)
		}

		for _, old := range d.Legacy {
			if strings.Contains(strings.ToLower(old), "boreal") {
				t.Errorf("%q still claims to replace %q", d.ID, old)
			}
		}
	}

	sql := readMigration(t)

	if !retired(sql, "staff", "borealis.*") {
		t.Error("the migration should list borealis.* as retired")
	}

	if strings.Contains(sql, "use_borealis") {
		t.Error("the migration still writes use_borealis into the database")
	}
}
