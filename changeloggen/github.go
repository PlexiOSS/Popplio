package changeloggen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"popplio/state"
)

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type FileDiff struct {
	Filename string
	Status   string
	Patch    string
}

type FileStats struct {
	FilesChanged int
	Additions    int
	Deletions    int
}

type CompareResult struct {
	PRs            []PullRequest
	Files          []FileDiff
	CommitMessages []string
	Stats          FileStats
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

var mergeCommitPR = regexp.MustCompile(`\(#(\d+)\)\s*$|Merge pull request #(\d+)`)

type compareResponse struct {
	Commits []struct {
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"commits"`
	Files []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Patch     string `json:"patch"`
	} `json:"files"`
}

func githubRequest(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return fmt.Errorf("failed to build GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Popplio-Changeloggen")

	if token := state.Config.Meta.GithubToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("GitHub request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return fmt.Errorf("GitHub returned status %d for %s: %s", resp.StatusCode, url, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return nil
}

func Compare(ctx context.Context, owner, repoName, base, head string) (CompareResult, error) {
	var cmp compareResponse

	compareURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/compare/%s...%s", owner, repoName, base, head)

	if err := githubRequest(ctx, compareURL, &cmp); err != nil {
		return CompareResult{}, err
	}

	stats := FileStats{FilesChanged: len(cmp.Files)}

	for _, f := range cmp.Files {
		stats.Additions += f.Additions
		stats.Deletions += f.Deletions
	}

	seen := make(map[int]struct{})
	var numbers []int

	for _, c := range cmp.Commits {
		matches := mergeCommitPR.FindStringSubmatch(c.Commit.Message)

		if matches == nil {
			continue
		}

		numStr := matches[1]

		if numStr == "" {
			numStr = matches[2]
		}

		var num int

		if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
			continue
		}

		if _, ok := seen[num]; ok {
			continue
		}

		seen[num] = struct{}{}
		numbers = append(numbers, num)
	}

	prs := make([]PullRequest, 0, len(numbers))

	for _, num := range numbers {
		var pr PullRequest

		prURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repoName, num)

		if err := githubRequest(ctx, prURL, &pr); err != nil {
			continue
		}

		prs = append(prs, pr)
	}

	result := CompareResult{PRs: prs, Stats: stats}

	if len(prs) > 0 {
		return result, nil
	}

	for _, f := range cmp.Files {
		result.Files = append(result.Files, FileDiff{Filename: f.Filename, Status: f.Status, Patch: f.Patch})
	}

	for _, c := range cmp.Commits {
		message := strings.TrimSpace(strings.SplitN(c.Commit.Message, "\n", 2)[0])

		if message == "" || strings.HasPrefix(message, "Merge ") {
			continue
		}

		result.CommitMessages = append(result.CommitMessages, message)
	}

	return result, nil
}
