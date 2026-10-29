// Copyright (C) 2026 NodeByte LTD

package changeloggen

type repo struct {
	Owner          string
	Repo           string
	ChangelogPaths []string
}

var projectRepos = map[string]repo{
	"popplio":  {Owner: "PlexiOSS", Repo: "Popplio"},
	"omniplex": {Owner: "PlexiOSS", Repo: "Omniplex", ChangelogPaths: []string{"apps/web/CHANGELOG.md", "CHANGELOG.md"}},
	"keel":     {Owner: "PlexiOSS", Repo: "Keel"},
}

func RepoFor(project string) (owner, repoName string, ok bool) {
	r, ok := projectRepos[project]
	return r.Owner, r.Repo, ok
}

func ChangelogPathsFor(project string) []string {
	if r, ok := projectRepos[project]; ok && len(r.ChangelogPaths) > 0 {
		return r.ChangelogPaths
	}

	return []string{"CHANGELOG.md"}
}
