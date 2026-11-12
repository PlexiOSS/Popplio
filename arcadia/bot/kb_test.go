package bot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"popplio/config"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
)

func withFrontend(t *testing.T) {
	prev := state.Config
	state.Config = &config.Config{Sites: config.Sites{Frontend: "https://example.com"}}
	t.Cleanup(func() { state.Config = prev })
}

func testKb() *kbIndex {
	return &kbIndex{
		Categories: []kbCategory{
			{Slug: "bots", Title: "Bots", Description: "Rules for bots"},
			{Slug: "voting", Title: "Voting", Description: "How voting works"},
		},
		Common: []string{"bots/getting-approved", "voting/missing"},
		Articles: []kbArticle{
			{Slug: "adding-a-bot", Category: "bots", CategoryTitle: "Bots", Title: "Adding a Bot", Description: "Submit your bot for review.", Sections: []string{"Before you start"}},
			{Slug: "getting-approved", Category: "bots", CategoryTitle: "Bots", Title: "Getting Your Bot Approved", Description: "How bot review works.", Summary: strings.Repeat("Reviewers check commands. ", 30), Sections: []string{"How review works", "Denials"}},
			{Slug: "auto-vote", Category: "voting", CategoryTitle: "Voting", Title: "Auto-vote", Description: "Vote automatically every day.", Sections: []string{"Pausing"}},
		},
	}
}

func TestKbSearchRanksTitleMatchesFirst(t *testing.T) {
	k := testKb()

	got := k.search("how do I get my bot approved", 5)

	if len(got) == 0 || got[0].Slug != "getting-approved" {
		t.Fatalf("expected getting-approved first, got %+v", got)
	}

	if got := k.search("auto-vote", 5); len(got) == 0 || got[0].Slug != "auto-vote" {
		t.Fatalf("expected auto-vote first, got %+v", got)
	}

	if got := k.search("the and of", 5); got != nil {
		t.Fatalf("stop words alone should match nothing, got %+v", got)
	}

	if got := k.search("pausing", 5); len(got) != 1 || got[0].Slug != "auto-vote" {
		t.Fatalf("section headings should be searchable, got %+v", got)
	}
}

func TestKbCommonSkipsMissingArticles(t *testing.T) {
	got := testKb().common()

	if len(got) != 1 || got[0].Slug != "getting-approved" {
		t.Fatalf("got %+v", got)
	}
}

func TestKbViewsFitDiscordLimits(t *testing.T) {
	withFrontend(t)

	k := testKb()
	a, _ := k.article("bots/getting-approved")

	for name, v := range map[string]kbView{
		"article":  kbArticleView(k, a, nil, ""),
		"search":   kbArticleView(k, a, k.Articles, "Other matches"),
		"category": kbCategoryView(k, k.Categories[0]),
		"home":     kbHomeView(k),
	} {
		if utf8.RuneCountInString(v.content) > 2000 {
			t.Errorf("%s: content is %d characters", name, utf8.RuneCountInString(v.content))
		}

		if len(v.components) > 5 {
			t.Errorf("%s: %d action rows", name, len(v.components))
		}

		msg := v.create()

		if msg.AllowedMentions == nil || len(msg.AllowedMentions.Parse) != 0 {
			t.Errorf("%s: replies must not ping anyone", name)
		}

		if msg.Flags&discord.MessageFlagSuppressEmbeds == 0 {
			t.Errorf("%s: links should not unfurl", name)
		}
	}
}

func TestKbArticleViewOffersTheRestOfItsCategory(t *testing.T) {
	withFrontend(t)

	k := testKb()
	a, _ := k.article("bots/getting-approved")
	v := kbArticleView(k, a, nil, "")

	if !strings.Contains(v.content, "**In this article:** How review works · Denials") {
		t.Fatalf("missing sections: %s", v.content)
	}

	row, ok := v.components[1].(discord.ActionRowComponent)

	if !ok {
		t.Fatalf("second row is %T", v.components[1])
	}

	menu, ok := row.Components[0].(discord.StringSelectMenuComponent)

	if !ok || menu.CustomID != kbArticleID || menu.Placeholder != "More in Bots" || len(menu.Options) != 2 {
		t.Fatalf("unexpected menu %+v", row.Components[0])
	}
}

func TestKbSafeStripsMarkdownAndMentions(t *testing.T) {
	if got := kbSafe("**@everyone** `hi`"); got != "everyone hi" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadKbFallsBackToStaleIndex(t *testing.T) {
	prevFetch, prevCached, prevAt, prevLog := kbFetch, kbCached, kbFetched, state.Logger
	t.Cleanup(func() { kbFetch, kbCached, kbFetched, state.Logger = prevFetch, prevCached, prevAt, prevLog })
	state.Logger = zap.NewNop()

	stale := testKb()
	kbCached, kbFetched = stale, time.Now().Add(-time.Hour)
	kbFetch = func(context.Context) (*kbIndex, error) { return nil, errors.New("down") }

	got, err := loadKb(context.Background())

	if err != nil || got != stale {
		t.Fatalf("expected the stale index, got %v %v", got, err)
	}

	kbCached = nil

	if _, err := loadKb(context.Background()); err == nil {
		t.Fatal("expected an error with nothing cached")
	}
}

func TestVoteBoardView(t *testing.T) {
	withFrontend(t)

	v := voteBoardView(voteBoardBots, []voteBoardRow{
		{name: "Petto", url: "https://example.com/bots/1", votes: 170},
		{name: "**Bold** @everyone", url: "https://example.com/bots/2", votes: 1},
	})

	if !strings.Contains(v.content, "🥇 **[Petto](<https://example.com/bots/1>)** · 170 votes") {
		t.Fatalf("unexpected first row: %s", v.content)
	}

	if !strings.Contains(v.content, "**[Bold everyone](<https://example.com/bots/2>)** · 1 vote") {
		t.Fatalf("names should be cleaned and votes pluralised: %s", v.content)
	}

	row := v.components[0].(discord.ActionRowComponent)
	current := row.Components[0].(discord.ButtonComponent)

	if !current.Disabled || current.Style != discord.ButtonStylePrimary {
		t.Fatalf("the shown board's button should be selected and disabled: %+v", current)
	}

	if empty := voteBoardView(voteBoardVoters, nil); !strings.Contains(empty.content, "No votes yet this month") {
		t.Fatalf("empty board: %s", empty.content)
	}
}
