package utils

import (
	"errors"
	"testing"
)

func TestGetDiscordWebhookInfoRecognisesEveryValidPrefix(t *testing.T) {
	prefixes := []string{
		"https://discordapp.com/",
		"https://discord.com/",
		"https://canary.discord.com/",
		"https://ptb.discord.com/",
	}

	for _, p := range prefixes {
		url := p + "api/webhooks/123/abc"

		gotPrefix, err := GetDiscordWebhookInfo(url)

		if err != nil {
			t.Errorf("GetDiscordWebhookInfo(%q) error = %v, want nil", url, err)
		}

		if gotPrefix != p {
			t.Errorf("GetDiscordWebhookInfo(%q) prefix = %q, want %q", url, gotPrefix, p)
		}
	}
}

func TestGetDiscordWebhookInfoRejectsDiscordURLWithoutWebhooksPath(t *testing.T) {
	url := "https://discord.com/api/channels/123"

	prefix, err := GetDiscordWebhookInfo(url)

	if !errors.Is(err, ErrNotActuallyWebhook) {
		t.Errorf("expected ErrNotActuallyWebhook, got %v", err)
	}

	if prefix != "https://discord.com/" {
		t.Errorf("prefix = %q, want the matched prefix even on error", prefix)
	}
}

func TestGetDiscordWebhookInfoIgnoresUnrelatedURLs(t *testing.T) {
	urls := []string{
		"https://example.com/webhooks/123",
		"https://not-discord.com/api/webhooks/123",
		"",
		"not a url",
	}

	for _, url := range urls {
		prefix, err := GetDiscordWebhookInfo(url)

		if err != nil {
			t.Errorf("GetDiscordWebhookInfo(%q) error = %v, want nil", url, err)
		}

		if prefix != "" {
			t.Errorf("GetDiscordWebhookInfo(%q) prefix = %q, want empty", url, prefix)
		}
	}
}
