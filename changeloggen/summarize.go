package changeloggen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

const systemPrompt = `You are writing a changelog entry for end users of a software product -- not developers. Given a list of merged pull request titles/descriptions and overall diff stats, categorize the changes into Added, Updated, Fixed, and Removed. Each bullet should be one short, plain-language sentence describing what changed from a USER's perspective -- never mention PR numbers, internal file names, or implementation details unless they're the actual user-facing feature. Skip purely internal changes (refactors, test additions, CI/tooling, dependency bumps) entirely unless they fix a user-visible bug. Also write a one-sentence "extra_description" summarizing the release's overall theme, or an empty string if there isn't one.

Respond with ONLY a JSON object of this exact shape:
{"added": ["..."], "updated": ["..."], "fixed": ["..."], "removed": ["..."], "extra_description": "..."}`

// Summarize turns raw PR data into a Draft. When Meta.OpenAIAPIKey is unset
// it falls back to bucketing PR titles by a title-prefix heuristic, verbatim
// -- rougher output, but the feature still works rather than erroring out,
// same contract as moderation.CheckText.
func Summarize(ctx context.Context, prs []PullRequest, stats FileStats) (Draft, error) {
	if len(prs) == 0 {
		return Draft{}, nil
	}

	apiKey := state.Config.Meta.OpenAIAPIKey

	if apiKey == "" {
		return heuristicDraft(prs), nil
	}

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

	reqBody, err := json.Marshal(chatRequest{
		Model:          "gpt-4o-mini",
		ResponseFormat: map[string]string{"type": "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: b.String()},
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
		return Draft{}, fmt.Errorf("summarize endpoint returned status %d", resp.StatusCode)
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

// heuristicDraft buckets PR titles by a simple prefix match when no OpenAI
// key is configured. Titles are used verbatim -- no rewriting.
func heuristicDraft(prs []PullRequest) Draft {
	var draft Draft

	for _, pr := range prs {
		title := strings.TrimSpace(pr.Title)
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
