package changeloggen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"popplio/state"
)

// Draft is a not-yet-saved changelog entry body -- everything CreateEntry
// needs except project/version/prerelease/published, which the admin sets.
type Draft struct {
	Added            []string `json:"added"`
	Updated          []string `json:"updated"`
	Fixed            []string `json:"fixed"`
	Removed          []string `json:"removed"`
	ExtraDescription string   `json:"extra_description"`
}

var summarizeHTTPClient = &http.Client{Timeout: 45 * time.Second}

// maxBodyChars bounds how much of a PR's body gets sent to the model --
// long bodies (design docs, screenshots-as-markdown) cost tokens without
// adding much signal for a one-line changelog bullet.
const maxBodyChars = 500

// maxPatchChars bounds how much of one file's diff gets sent to the model
// in the no-PRs fallback -- large generated/vendored files would otherwise
// dominate the prompt budget for no summarization value.
const maxPatchChars = 1500

// maxTotalDiffChars caps the combined diff content sent across all files,
// so a release touching hundreds of files still fits a reasonable prompt.
const maxTotalDiffChars = 12000

type chatRequest struct {
	Model          string            `json:"model"`
	ResponseFormat map[string]string `json:"response_format"`
	Messages       []chatMessage     `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

const prSystemPrompt = `You are writing a changelog entry for end users of a software product -- not developers. Given a list of merged pull request titles/descriptions and overall diff stats, categorize the changes into Added, Updated, Fixed, and Removed. Each bullet should be one short, plain-language sentence describing what changed from a USER's perspective -- never mention PR numbers, internal file names, or implementation details unless they're the actual user-facing feature. Skip purely internal changes (refactors, test additions, CI/tooling, dependency bumps) entirely unless they fix a user-visible bug. Also write a one-sentence "extra_description" summarizing the release's overall theme, or an empty string if there isn't one.

Respond with ONLY a JSON object of this exact shape:
{"added": ["..."], "updated": ["..."], "fixed": ["..."], "removed": ["..."], "extra_description": "..."}`

const diffSystemPrompt = `You are writing a changelog entry for end users of a software product -- not developers. This repository pushes commits straight to its branch instead of merging pull requests, so there are no PR titles to summarize -- instead you're given the raw commit subjects (often terse or unhelpful, e.g. "fix some stuff") AND the actual code diff for each changed file. Read the diff itself to figure out what really changed; do not just rephrase the commit messages, since they're often misleading or too vague to use directly. Categorize what you find into Added, Updated, Fixed, and Removed. Each bullet should be one short, plain-language sentence describing what changed from a USER's perspective -- never mention file names, function names, or line numbers unless that's genuinely the most useful way to describe a developer-facing library change. Skip purely internal changes (formatting, comments, tests, CI/tooling) entirely unless they fix a user-visible bug. Also write a one-sentence "extra_description" summarizing the release's overall theme, or an empty string if there isn't one.

Respond with ONLY a JSON object of this exact shape:
{"added": ["..."], "updated": ["..."], "fixed": ["..."], "removed": ["..."], "extra_description": "..."}`

// Summarize turns a GitHub compare result into a Draft. When
// Meta.OpenAIAPIKey is unset it falls back to a title/commit-bucketing
// heuristic -- rougher output, but the feature still works rather than
// erroring out, same contract as moderation.CheckText.
func Summarize(ctx context.Context, cmp CompareResult) (Draft, error) {
	apiKey := state.Config.Meta.OpenAIAPIKey

	if len(cmp.PRs) > 0 {
		if apiKey == "" {
			return heuristicDraftFromTitles(titlesOf(cmp.PRs)), nil
		}

		return callChat(ctx, apiKey, prSystemPrompt, prPrompt(cmp.PRs, cmp.Stats))
	}

	if len(cmp.Files) == 0 && len(cmp.CommitMessages) == 0 {
		return Draft{}, nil
	}

	if apiKey == "" {
		// No LLM available to actually read the diff -- bucket the raw
		// commit subjects instead, same as the PR-title heuristic.
		return heuristicDraftFromTitles(cmp.CommitMessages), nil
	}

	return callChat(ctx, apiKey, diffSystemPrompt, diffPrompt(cmp.CommitMessages, cmp.Files, cmp.Stats))
}

func titlesOf(prs []PullRequest) []string {
	titles := make([]string, len(prs))

	for i, pr := range prs {
		titles[i] = pr.Title
	}

	return titles
}

func prPrompt(prs []PullRequest, stats FileStats) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Diff stats: %d files changed, +%d/-%d lines.\n\nMerged pull requests:\n",
		stats.FilesChanged, stats.Additions, stats.Deletions)

	for _, pr := range prs {
		body := pr.Body

		if len(body) > maxBodyChars {
			body = body[:maxBodyChars] + "..."
		}

		fmt.Fprintf(&b, "- %s\n", pr.Title)

		if body != "" {
			fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(body, "\n", " "))
		}
	}

	return b.String()
}

func diffPrompt(commitMessages []string, files []FileDiff, stats FileStats) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Diff stats: %d files changed, +%d/-%d lines.\n\n", stats.FilesChanged, stats.Additions, stats.Deletions)

	if len(commitMessages) > 0 {
		b.WriteString("Commit subjects (for context only -- may be unhelpful, verify against the diff below):\n")

		for _, m := range commitMessages {
			fmt.Fprintf(&b, "- %s\n", m)
		}

		b.WriteString("\n")
	}

	b.WriteString("File changes:\n")

	remaining := maxTotalDiffChars

	for _, f := range files {
		if remaining <= 0 {
			b.WriteString("\n(remaining files omitted for space)\n")
			break
		}

		fmt.Fprintf(&b, "\n--- %s (%s) ---\n", f.Filename, f.Status)

		patch := f.Patch

		if patch == "" {
			b.WriteString("(no diff available -- binary or too large)\n")
			continue
		}

		if len(patch) > maxPatchChars {
			patch = patch[:maxPatchChars] + "\n...(truncated)"
		}

		if len(patch) > remaining {
			patch = patch[:remaining] + "\n...(truncated)"
		}

		b.WriteString(patch)
		b.WriteString("\n")
		remaining -= len(patch)
	}

	return b.String()
}

func callChat(ctx context.Context, apiKey, systemPrompt, userContent string) (Draft, error) {
	reqBody, err := json.Marshal(chatRequest{
		Model:          "gpt-4o-mini",
		ResponseFormat: map[string]string{"type": "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	})

	if err != nil {
		return Draft{}, fmt.Errorf("failed to marshal summarize request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))

	if err != nil {
		return Draft{}, fmt.Errorf("failed to build summarize request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := summarizeHTTPClient.Do(req)

	if err != nil {
		return Draft{}, fmt.Errorf("summarize request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Same reasoning as githubRequest: a bare status code can't tell a
		// per-minute rate limit apart from an exhausted billing quota, and
		// those need different fixes.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return Draft{}, fmt.Errorf("summarize endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed chatResponse

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Draft{}, fmt.Errorf("failed to decode summarize response: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return Draft{}, fmt.Errorf("summarize response had no choices")
	}

	var draft Draft

	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &draft); err != nil {
		return Draft{}, fmt.Errorf("failed to decode summarized draft: %w", err)
	}

	return draft, nil
}

// heuristicDraftFromTitles buckets plain title/subject strings (PR titles or
// commit subjects) by a simple prefix match when no OpenAI key is
// configured. Used verbatim -- no rewriting.
func heuristicDraftFromTitles(titles []string) Draft {
	var draft Draft

	for _, raw := range titles {
		title := strings.TrimSpace(raw)
		lower := strings.ToLower(title)

		switch {
		case hasAnyPrefix(lower, "fix", "bug"):
			draft.Fixed = append(draft.Fixed, title)
		case hasAnyPrefix(lower, "feat", "add", "new"):
			draft.Added = append(draft.Added, title)
		case hasAnyPrefix(lower, "remove", "delete", "drop"):
			draft.Removed = append(draft.Removed, title)
		default:
			draft.Updated = append(draft.Updated, title)
		}
	}

	return draft
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}

	return false
}
