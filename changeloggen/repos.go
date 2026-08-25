// Package changeloggen generates a draft changelog entry (Added/Updated/
// Fixed/Removed + a short description) from the merged PRs between two git
// refs on GitHub, optionally cleaned up by OpenAI into end-user-facing
// language. Nothing here persists anything -- callers get a Draft back and
// decide whether to save it.
package changeloggen

// repo is the GitHub owner/repo a project's changelog entries diff against.
type repo struct {
	Owner string
	Repo  string
}

var projectRepos = map[string]repo{
	"popplio":  {Owner: "PlexiOSS", Repo: "Popplio"},
	"omniplex": {Owner: "PlexiOSS", Repo: "Omniplex"},
	"keel":     {Owner: "PlexiOSS", Repo: "Keel"},
}

// RepoFor returns the GitHub owner/repo for a changelog project, or false if
// the project isn't recognized.
func RepoFor(project string) (owner, repoName string, ok bool) {
	r, ok := projectRepos[project]
	return r.Owner, r.Repo, ok
}
