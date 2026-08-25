package changeloggen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"popplio/state"
)

// PullRequest is the subset of a GitHub PR's data the summarizer needs.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// FileStats summarizes the shape of a diff -- not stored anywhere, only
// handed to the summarizer as scale context ("this was a big release").
type FileStats struct {
	FilesChanged int
	Additions    int
	Deletions    int
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

// mergeCommitPR matches GitHub's default merge-commit message formats:
// squash-merges ("Title (#123)") and regular merges ("Merge pull request
// #123 from owner/branch").
var mergeCommitPR = regexp.MustCompile(`\(#(\d+)\)\s*$|Merge pull request #(\d+)`)

type compareResponse struct {
	Commits []struct {
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"commits"`
	Files []struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"files"`
}

func githubRequest(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return fmt.Errorf("failed to build GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if token := state.Config.Meta.GithubToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("GitHub request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned status %d for %s", resp.StatusCode, url)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return nil
}

// ComparePRs diffs base...head on GitHub and returns the merged PRs it can
// identify from the commit list, plus overall file-change stats.
func ComparePRs(ctx context.Context, owner, repoName, base, head string) ([]PullRequest, FileStats, error) {
	var cmp compareResponse

	compareURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/compare/%s...%s", owner, repoName, base, head)

	if err := githubRequest(ctx, compareURL, &cmp); err != nil {
		return nil, FileStats{}, err
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
			// One PR failing to fetch (e.g. it was since deleted) shouldn't
			// sink the whole draft -- skip it.
			continue
		}

		prs = append(prs, pr)
	}

	return prs, stats, nil
}
