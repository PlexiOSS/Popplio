package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

type kbCategory struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type kbArticle struct {
	Slug          string   `json:"slug"`
	Category      string   `json:"category"`
	CategoryTitle string   `json:"categoryTitle"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Summary       string   `json:"summary"`
	Sections      []string `json:"sections"`
}

func (a kbArticle) id() string {
	return a.Category + "/" + a.Slug
}

func (a kbArticle) url() string {
	return fmt.Sprintf("%s/kb/%s/%s", frontendURL(), a.Category, a.Slug)
}

type kbIndex struct {
	Categories []kbCategory `json:"categories"`
	Common     []string     `json:"common"`
	Articles   []kbArticle  `json:"articles"`
}

func (k *kbIndex) article(id string) (kbArticle, bool) {
	for _, a := range k.Articles {
		if a.id() == id {
			return a, true
		}
	}

	return kbArticle{}, false
}

func (k *kbIndex) category(slug string) (kbCategory, bool) {
	for _, c := range k.Categories {
		if c.Slug == slug {
			return c, true
		}
	}

	return kbCategory{}, false
}

func (k *kbIndex) inCategory(slug string) []kbArticle {
	var out []kbArticle

	for _, a := range k.Articles {
		if a.Category == slug {
			out = append(out, a)
		}
	}

	return out
}

func (k *kbIndex) common() []kbArticle {
	var out []kbArticle

	for _, id := range k.Common {
		if a, ok := k.article(id); ok {
			out = append(out, a)
		}
	}

	return out
}

const kbTTL = 10 * time.Minute

var (
	kbMu      sync.Mutex
	kbCached  *kbIndex
	kbFetched time.Time
	kbHTTP    = &http.Client{Timeout: 5 * time.Second}
	kbFetch   = fetchKbIndex
)

func fetchKbIndex(ctx context.Context) (*kbIndex, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, frontendURL()+"/api/kb", nil)

	if err != nil {
		return nil, err
	}

	resp, err := kbHTTP.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("knowledge base index returned %d", resp.StatusCode)
	}

	var idx kbIndex

	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, err
	}

	return &idx, nil
}

func loadKb(ctx context.Context) (*kbIndex, error) {
	kbMu.Lock()
	defer kbMu.Unlock()

	if kbCached != nil && time.Since(kbFetched) < kbTTL {
		return kbCached, nil
	}

	idx, err := kbFetch(ctx)

	if err != nil {
		if kbCached != nil {
			state.Logger.Warn("Failed to refresh the knowledge base index, using the cached one", zap.Error(err))
			return kbCached, nil
		}

		return nil, err
	}

	kbCached, kbFetched = idx, time.Now()
	return idx, nil
}

var kbStopWords = func() map[string]struct{} {
	m := map[string]struct{}{}

	for _, w := range strings.Fields("a an and are as at be but by can do does for from get got have how i i'm im in into is it its me my no not of on or our so that the their them then there this to too up us was we what when where which who why will with you your help please cant can't doesnt doesn't dont don't wont won't isnt isn't") {
		m[strings.ReplaceAll(w, "'", "")] = struct{}{}
	}

	return m
}()

var kbSplit = regexp.MustCompile(`[^a-z0-9']+`)
var kbSuffix = regexp.MustCompile(`(ing|ed|es|s)$`)

func kbWords(text string) []string {
	var out []string

	for _, w := range kbSplit.Split(strings.ToLower(text), -1) {
		w = strings.ReplaceAll(w, "'", "")

		if len(w) < 2 {
			continue
		}

		if _, stop := kbStopWords[w]; stop {
			continue
		}

		if s := kbSuffix.ReplaceAllString(w, ""); s != "" {
			w = s
		}

		out = append(out, w)
	}

	return out
}

func kbSet(text string) map[string]struct{} {
	m := map[string]struct{}{}

	for _, w := range kbWords(text) {
		m[w] = struct{}{}
	}

	return m
}

func (k *kbIndex) search(query string, limit int) []kbArticle {
	terms := kbWords(query)

	if len(terms) == 0 {
		return nil
	}

	slices.Sort(terms)
	terms = slices.Compact(terms)
	phrase := strings.ToLower(strings.TrimSpace(query))

	type scored struct {
		article kbArticle
		score   int
		order   int
	}

	var results []scored

	for i, a := range k.Articles {
		title := kbSet(a.Title)
		description := kbSet(a.Description)
		keywords := kbSet(strings.Join(a.Sections, " ") + " " + a.CategoryTitle)
		score := 0

		if strings.Contains(strings.ToLower(a.Title), phrase) {
			score += 6
		}

		for _, t := range terms {
			if _, ok := title[t]; ok {
				score += 3
			} else if _, ok := description[t]; ok {
				score += 2
			} else if _, ok := keywords[t]; ok {
				score++
			}
		}

		if score > 0 {
			results = append(results, scored{a, score, i})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}

		return results[i].order < results[j].order
	})

	out := make([]kbArticle, 0, min(limit, len(results)))

	for _, r := range results {
		if len(out) == limit {
			break
		}

		out = append(out, r.article)
	}

	return out
}

func kbClip(s string, n int) string {
	r := []rune(s)

	if len(r) <= n {
		return s
	}

	return strings.TrimSpace(string(r[:n-1])) + "…"
}

var kbMarkdown = strings.NewReplacer("*", "", "_", "", "~", "", "`", "", "|", "", ">", "", "#", "", "[", "", "]", "", "@", "")

func kbSafe(s string) string {
	return kbClip(strings.TrimSpace(kbMarkdown.Replace(s)), 80)
}

const (
	kbArticleID  = "kb:a"
	kbCategoryID = "kb:c"
)

func kbArticleOptions(articles []kbArticle, current string) []discord.StringSelectMenuOption {
	opts := make([]discord.StringSelectMenuOption, 0, min(len(articles), 25))

	for _, a := range articles {
		if len(opts) == 25 {
			break
		}

		opt := discord.NewStringSelectMenuOption(kbClip(a.Title, 100), a.id())

		if a.Description != "" {
			opt = opt.WithDescription(kbClip(a.Description, 100))
		}

		opts = append(opts, opt.WithDefault(a.id() == current))
	}

	return opts
}

func kbCategoryMenu(k *kbIndex, current string) discord.ContainerComponent {
	opts := make([]discord.StringSelectMenuOption, 0, len(k.Categories))

	for _, c := range k.Categories {
		if len(opts) == 25 {
			break
		}

		opts = append(opts, discord.NewStringSelectMenuOption(c.Title, c.Slug).
			WithDescription(kbClip(c.Description, 100)).
			WithDefault(c.Slug == current))
	}

	return discord.NewActionRow(discord.NewStringSelectMenu(kbCategoryID, "Browse a category", opts...))
}

type kbView struct {
	content    string
	components []discord.ContainerComponent
}

func (v kbView) create() discord.MessageCreate {
	return discord.MessageCreate{
		Content:         v.content,
		Components:      v.components,
		AllowedMentions: &discord.AllowedMentions{},
		Flags:           discord.MessageFlagSuppressEmbeds,
	}
}

func (v kbView) update() discord.MessageUpdate {
	return discord.MessageUpdate{
		Content:         &v.content,
		Components:      &v.components,
		AllowedMentions: &discord.AllowedMentions{},
	}
}

func kbArticleView(k *kbIndex, a kbArticle, others []kbArticle, othersLabel string) kbView {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## %s\n-# %s\n", a.Title, a.CategoryTitle)

	if a.Description != "" {
		fmt.Fprintf(&sb, "%s\n", a.Description)
	}

	if a.Summary != "" && a.Summary != a.Description {
		fmt.Fprintf(&sb, "\n%s\n", a.Summary)
	}

	if len(a.Sections) > 0 {
		fmt.Fprintf(&sb, "\n**In this article:** %s\n", strings.Join(a.Sections, " · "))
	}

	content := kbClip(sb.String(), 1900)

	components := []discord.ContainerComponent{
		discord.NewActionRow(discord.NewLinkButton("Read the article", a.url())),
	}

	if othersLabel == "" {
		others = k.inCategory(a.Category)
		othersLabel = "More in " + a.CategoryTitle
	}

	if len(others) > 1 || (len(others) == 1 && others[0].id() != a.id()) {
		components = append(components, discord.NewActionRow(
			discord.NewStringSelectMenu(kbArticleID, kbClip(othersLabel, 150), kbArticleOptions(others, a.id())...),
		))
	}

	components = append(components, kbCategoryMenu(k, ""))

	return kbView{content: content, components: components}
}

func kbCategoryView(k *kbIndex, c kbCategory) kbView {
	articles := k.inCategory(c.Slug)

	var sb strings.Builder

	fmt.Fprintf(&sb, "## %s\n%s\n\n", c.Title, c.Description)

	for _, a := range articles {
		line := fmt.Sprintf("- [%s](<%s>)\n", a.Title, a.url())

		if sb.Len()+len(line) > 1800 {
			sb.WriteString("- …and more on the website\n")
			break
		}

		sb.WriteString(line)
	}

	components := []discord.ContainerComponent{}

	if len(articles) > 0 {
		components = append(components, discord.NewActionRow(
			discord.NewStringSelectMenu(kbArticleID, "Open an article", kbArticleOptions(articles, "")...),
		))
	}

	components = append(components,
		kbCategoryMenu(k, c.Slug),
		discord.NewActionRow(discord.NewLinkButton("Open "+c.Title, fmt.Sprintf("%s/kb/%s", frontendURL(), c.Slug))),
	)

	return kbView{content: sb.String(), components: components}
}

func kbHomeView(k *kbIndex) kbView {
	var sb strings.Builder

	sb.WriteString("## Knowledge Base\nPick a category below, or run `/kb` with a topic and I'll find the right article.\n")

	if common := k.common(); len(common) > 0 {
		sb.WriteString("\n**Common questions**\n")

		for _, a := range common {
			fmt.Fprintf(&sb, "- [%s](<%s>)\n", a.Title, a.url())
		}
	}

	components := []discord.ContainerComponent{kbCategoryMenu(k, "")}

	if common := k.common(); len(common) > 0 {
		components = append(components, discord.NewActionRow(
			discord.NewStringSelectMenu(kbArticleID, "Open a common question", kbArticleOptions(common, "")...),
		))
	}

	components = append(components, discord.NewActionRow(discord.NewLinkButton("Open the Knowledge Base", frontendURL()+"/kb")))

	return kbView{content: sb.String(), components: components}
}

func cmdKb() *Command {
	return &Command{
		Name:        "kb",
		Category:    "Help",
		Description: "Find an answer in the Knowledge Base",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:         "topic",
				Description:  "What you need help with",
				Autocomplete: true,
			},
		},
		Autocomplete: func(c *Ctx, value string) []discord.AutocompleteChoice {
			k, err := loadKb(c.Context)

			if err != nil {
				return nil
			}

			var articles []kbArticle

			if strings.TrimSpace(value) == "" {
				articles = k.common()
			} else {
				articles = k.search(value, 25)
			}

			choices := make([]discord.AutocompleteChoice, 0, len(articles))

			for _, a := range articles {
				choices = append(choices, discord.AutocompleteChoiceString{
					Name:  kbClip(a.Title+" · "+a.CategoryTitle, 100),
					Value: a.id(),
				})
			}

			return choices
		},
		Run: func(c *Ctx) error {
			k, err := loadKb(c.Context)

			if err != nil {
				state.Logger.Error("Failed to load the knowledge base index", zap.Error(err))
				return c.Say(fmt.Sprintf("I couldn't reach the Knowledge Base just now. You can browse it at <%s/kb>.", frontendURL()))
			}

			topic := strings.TrimSpace(c.Option("topic", 0))

			if topic == "" {
				return c.Send(kbHomeView(k).create())
			}

			if a, ok := k.article(topic); ok {
				return c.Send(kbArticleView(k, a, nil, "").create())
			}

			matches := k.search(topic, 6)

			if len(matches) == 0 {
				return c.Send(kbView{
					content: fmt.Sprintf("I couldn't find an article about **%s**. Try different words, browse a category below, or open a ticket with `/ticket` and staff will help.", kbSafe(topic)),
					components: []discord.ContainerComponent{
						kbCategoryMenu(k, ""),
						discord.NewActionRow(discord.NewLinkButton("Open the Knowledge Base", frontendURL()+"/kb")),
					},
				}.create())
			}

			return c.Send(kbArticleView(k, matches[0], matches, "Other matches").create())
		},
	}
}

func onKbComponent(ctx context.Context, e *events.ComponentInteractionCreate) {
	id := e.Data.CustomID()
	values := e.StringSelectMenuInteractionData().Values

	if len(values) == 0 {
		_ = e.DeferUpdateMessage()
		return
	}

	k, err := loadKb(ctx)

	if err != nil {
		_ = e.CreateMessage(discord.MessageCreate{
			Content: fmt.Sprintf("I couldn't reach the Knowledge Base just now. You can browse it at <%s/kb>.", frontendURL()),
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	var view kbView

	switch id {
	case kbArticleID:
		a, ok := k.article(values[0])

		if !ok {
			_ = e.DeferUpdateMessage()
			return
		}

		view = kbArticleView(k, a, nil, "")
	case kbCategoryID:
		cat, ok := k.category(values[0])

		if !ok {
			_ = e.DeferUpdateMessage()
			return
		}

		view = kbCategoryView(k, cat)
	default:
		return
	}

	if meta := e.Message.InteractionMetadata; meta != nil && meta.User.ID != e.User().ID {
		msg := view.create()
		msg.Flags |= discord.MessageFlagEphemeral

		if err := e.CreateMessage(msg); err != nil {
			state.Logger.Error("Failed to send a knowledge base view", zap.Error(err))
		}

		return
	}

	if err := e.UpdateMessage(view.update()); err != nil {
		state.Logger.Error("Failed to update a knowledge base view", zap.Error(err))
	}
}
