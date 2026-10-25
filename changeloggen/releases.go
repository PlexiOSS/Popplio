// Copyright (C) 2026 NodeByte LTD

package changeloggen

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"popplio/state"
)

var ErrNoChangelog = errors.New("repository has no CHANGELOG.md")

type Release struct {
	Version string
	Date    string
	Body    string
}

const maxReleaseChars = 14000

var releaseHeading = regexp.MustCompile(`^## \[([^\]]+)\](?:\s*-\s*(\S+))?`)

func FetchChangelogFile(ctx context.Context, owner, repoName string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/CHANGELOG.md", owner, repoName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return "", fmt.Errorf("failed to build GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.raw+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Popplio-Changeloggen")

	if token := state.Config.Meta.GithubToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return "", fmt.Errorf("GitHub request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrNoChangelog
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return "", fmt.Errorf("GitHub returned status %d for %s: %s", resp.StatusCode, url, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if err != nil {
		return "", fmt.Errorf("failed to read CHANGELOG.md: %w", err)
	}

	return string(body), nil
}

func ParseReleases(markdown string) []Release {
	var releases []Release
	var current *Release
	var body strings.Builder

	flush := func() {
		if current != nil {
			current.Body = strings.TrimSpace(body.String())
			releases = append(releases, *current)
		}
		body.Reset()
	}

	for _, line := range strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = nil

			m := releaseHeading.FindStringSubmatch(line)

			if m != nil && !strings.EqualFold(m[1], "unreleased") {
				current = &Release{Version: m[1], Date: m[2]}
			}

			continue
		}

		if current != nil {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}

	flush()

	return releases
}

func CompareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)

	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}

	return 0
}

func versionParts(v string) [3]int {
	var out [3]int

	core := strings.SplitN(strings.TrimPrefix(v, "v"), "-", 2)[0]

	for i, part := range strings.SplitN(core, ".", 3) {
		n, err := strconv.Atoi(part)

		if err == nil && i < 3 {
			out[i] = n
		}
	}

	return out
}

const releaseSystemPrompt = `You are turning a developer-facing changelog section into a release note for end users of a website and its Discord bots -- not developers. The input is one version's section from a Keep a Changelog CHANGELOG.md, written for engineers: it explains root causes, file names, endpoints, config keys, migrations and implementation details. Rewrite it for users. Categorize what users will notice into Added, Updated, Fixed, Removed, and Security. Security is for fixes to vulnerabilities, data exposure, or access control, described without giving away how to exploit them. Each bullet is one short, plain-language sentence about what changed from a USER's perspective; fixes start with "Fixed". Never mention file names, function names, endpoints, config keys, migrations, table names, or library names. Merge related engineering entries into one user-facing bullet, and drop purely internal entries (refactors, logging, tooling, docs) unless they change something users see. Do not use em dashes. Also write a one-sentence "extra_description" summarizing the release's overall theme.

Respond with ONLY a JSON object of this exact shape:
{"added": ["..."], "updated": ["..."], "fixed": ["..."], "removed": ["..."], "security": ["..."], "extra_description": "..."}`

func SummarizeRelease(ctx context.Context, project string, release Release) (Draft, error) {
	body := release.Body

	if len(body) > maxReleaseChars {
		body = body[:maxReleaseChars] + "\n(truncated)"
	}

	if apiKey := state.Config.Meta.OpenAIAPIKey; apiKey != "" {
		prompt := fmt.Sprintf("Project: %s\nVersion: %s\n\n%s", project, release.Version, body)
		return callChat(ctx, apiKey, releaseSystemPrompt, prompt)
	}

	return heuristicDraftFromRelease(body), nil
}

func heuristicDraftFromRelease(body string) Draft {
	var draft Draft
	var section string
	var bullet strings.Builder

	flush := func() {
		text := strings.TrimSpace(bullet.String())
		bullet.Reset()

		if text == "" {
			return
		}

		if i := strings.Index(text, ". "); i > 0 {
			text = text[:i+1]
		}

		switch section {
		case "added":
			draft.Added = append(draft.Added, text)
		case "fixed":
			draft.Fixed = append(draft.Fixed, text)
		case "removed", "deprecated":
			draft.Removed = append(draft.Removed, text)
		case "security":
			draft.Security = append(draft.Security, text)
		default:
			draft.Updated = append(draft.Updated, text)
		}
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "### "):
			flush()
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))
		case strings.HasPrefix(line, "- "):
			flush()
			bullet.WriteString(strings.TrimPrefix(line, "- "))
		case trimmed != "" && bullet.Len() > 0:
			bullet.WriteString(" ")
			bullet.WriteString(trimmed)
		case trimmed == "":
			flush()
		}
	}

	flush()

	return draft
}
