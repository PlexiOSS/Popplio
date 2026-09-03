// Copyright (C) 2026 NodeByte LTD

package validators

import (
	"strings"
	"testing"

	"popplio/types"
)

func link(name, value string) types.Link {
	return types.Link{Name: name, Value: value}
}

func TestValidateExtraLinksAcceptsAWellFormedMix(t *testing.T) {
	links := []types.Link{
		link("Website", "https://example.com"),
		link("_internal", "anything at all"),
	}

	if err := ValidateExtraLinks(links); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateExtraLinksAcceptsEmpty(t *testing.T) {
	if err := ValidateExtraLinks(nil); err != nil {
		t.Errorf("expected no error for an empty list, got %v", err)
	}
}

func TestValidateExtraLinksRejectsTooManyLinksOverall(t *testing.T) {
	links := make([]types.Link, 21)
	for i := range links {
		links[i] = link("_a", "v")
	}

	if err := ValidateExtraLinks(links); err == nil {
		t.Error("expected an error for more than 20 links total")
	}
}

func TestValidateExtraLinksRejectsTooManyPublicLinks(t *testing.T) {
	links := make([]types.Link, 11)
	for i := range links {
		links[i] = link("Site", "https://example.com")
	}

	if err := ValidateExtraLinks(links); err == nil {
		t.Error("expected an error for more than 10 public links")
	}
}

func TestValidateExtraLinksRejectsTooManyPrivateLinks(t *testing.T) {
	links := make([]types.Link, 11)
	for i := range links {
		links[i] = link("_asset", "value")
	}

	if err := ValidateExtraLinks(links); err == nil {
		t.Error("expected an error for more than 10 private links")
	}
}

func TestValidateExtraLinksRejectsOversizedPublicFields(t *testing.T) {
	longName := strings.Repeat("a", 65)

	if err := ValidateExtraLinks([]types.Link{link(longName, "https://example.com")}); err == nil {
		t.Error("expected an error for a public link name over 64 chars")
	}

	longValue := "https://" + strings.Repeat("a", 512)

	if err := ValidateExtraLinks([]types.Link{link("Site", longValue)}); err == nil {
		t.Error("expected an error for a public link value over 512 chars")
	}
}

func TestValidateExtraLinksRejectsOversizedPrivateFields(t *testing.T) {
	longName := "_" + strings.Repeat("a", 512)

	if err := ValidateExtraLinks([]types.Link{link(longName, "v")}); err == nil {
		t.Error("expected an error for a private link name over 512 chars")
	}

	longValue := strings.Repeat("a", 8193)

	if err := ValidateExtraLinks([]types.Link{link("_a", longValue)}); err == nil {
		t.Error("expected an error for a private link value over 8192 chars")
	}
}

func TestValidateExtraLinksRejectsEmptyFields(t *testing.T) {
	cases := []types.Link{
		link("", "https://example.com"),
		link("   ", "https://example.com"),
		link("Site", ""),
		link("Site", "   "),
		link("_a", ""),
		link("", "_value"),
	}

	for _, l := range cases {
		if err := ValidateExtraLinks([]types.Link{l}); err == nil {
			t.Errorf("expected an error for blank field in %+v", l)
		}
	}
}

func TestValidateExtraLinksRequiresHTTPSForPublicLinks(t *testing.T) {
	if err := ValidateExtraLinks([]types.Link{link("Site", "http://example.com")}); err == nil {
		t.Error("expected an error for a non-HTTPS public link")
	}
}

func TestValidateExtraLinksAllowsNonHTTPSForPrivateLinks(t *testing.T) {
	if err := ValidateExtraLinks([]types.Link{link("_a", "not-a-url-at-all")}); err != nil {
		t.Errorf("private links should not require HTTPS, got %v", err)
	}
}

func TestValidateExtraLinksRejectsInvalidCharactersInName(t *testing.T) {
	cases := []string{"Site!", "Site<script>", "Site#1", "Site/path"}

	for _, name := range cases {
		if err := ValidateExtraLinks([]types.Link{link(name, "https://example.com")}); err == nil {
			t.Errorf("expected an error for invalid character in name %q", name)
		}
	}
}

func TestValidateExtraLinksAllowsAlphanumericSpaceDashUnderscoreInName(t *testing.T) {
	if err := ValidateExtraLinks([]types.Link{link("My Site-1_2", "https://example.com")}); err != nil {
		t.Errorf("expected valid chars to be accepted, got %v", err)
	}
}
