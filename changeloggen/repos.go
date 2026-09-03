// Copyright (C) 2026 NodeByte LTD

package changeloggen

type repo struct {
	Owner string
	Repo  string
}

var projectRepos = map[string]repo{
	"popplio":  {Owner: "PlexiOSS", Repo: "Popplio"},
	"omniplex": {Owner: "PlexiOSS", Repo: "Omniplex"},
	"keel":     {Owner: "PlexiOSS", Repo: "Keel"},
}

func RepoFor(project string) (owner, repoName string, ok bool) {
	r, ok := projectRepos[project]
	return r.Owner, r.Repo, ok
}
