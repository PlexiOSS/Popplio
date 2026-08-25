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

// PullRequest is the subset of a GitHub PR's data the summarizer needs.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// FileDiff is one changed file's patch from a GitHub compare -- used when
// there's no PR data to summarize from, so the model can read what actually
// changed in the code instead of guessing from a commit subject line alone.
type FileDiff struct {
	Filename string
	Status   string
	// Patch is the unified diff for this file. GitHub omits it for very
	// large or binary files, in which case this is empty.
	Patch string
}

// FileStats summarizes the shape of a diff -- handed to the summarizer as
// scale context ("this was a big release").
type FileStats struct {
	FilesChanged int
	Additions    int
	Deletions    int
}

// CompareResult is everything GitHub's compare API gave us about base...head.
type CompareResult struct {
	// PRs is populated only when commits in the range carry a recognizable
	// merge-commit message (squash or merge). Empty when the repo pushes
	// straight to the branch instead of merging PRs.
	PRs []PullRequest
	// Files and CommitMessages back the diff-based fallback when PRs is
	// empty -- the actual code changes and raw commit subjects, for the
	// summarizer to read directly rather than needing a PR title to exist.
	Files          []FileDiff
	CommitMessages []string
	Stats          FileStats
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
	// The GitHub API rejects requests with no User-Agent at all; Go's
	// default ("Go-http-client/1.x") normally satisfies this, but setting
	// our own is one less thing to guess about when diagnosing a failure.
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
		// GitHub's error body (rate-limited, bad credentials, SSO not
		// authorized, etc.) is the actual reason -- swallowing it just
		// forces re-diagnosing the same failure by hand every time.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return fmt.Errorf("GitHub returned status %d for %s: %s", resp.StatusCode, url, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return nil
}

// Compare diffs base...head on GitHub. When commits in the range carry
// recognizable merge-commit messages, it fetches the merged PRs behind
// them; otherwise it returns the raw per-file patches and commit subjects
// instead, so a repo that pushes straight to its branch still gets
// something the summarizer can work from.
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
			// One PR failing to fetch (e.g. it was since deleted) shouldn't
			// sink the whole draft -- skip it.
			continue
		}

		prs = append(prs, pr)
	}

	result := CompareResult{PRs: prs, Stats: stats}

	if len(prs) > 0 {
		return result, nil
	}

	// No merge-commit-shaped PRs found -- this repo pushes straight to the
	// branch. Hand the summarizer the actual diff instead of giving up.
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
