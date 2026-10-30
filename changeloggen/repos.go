// Copyright (C) 2026 NodeByte LTD

package changeloggen

import (
	"regexp"
	"strings"
)

type repo struct {
	Owner          string
	Repo           string
	ChangelogPaths []string
	PackagesDir    string
}

var projectRepos = map[string]repo{
	"popplio":  {Owner: "PlexiOSS", Repo: "Popplio"},
	"omniplex": {Owner: "PlexiOSS", Repo: "Omniplex", ChangelogPaths: []string{"apps/web/CHANGELOG.md", "CHANGELOG.md"}, PackagesDir: "packages"},
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

func PackagesDirFor(project string) string {
	return projectRepos[project].PackagesDir
}

var packageName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func ValidPackageName(name string) bool {
	return packageName.MatchString(name)
}

var packageLabels = map[string]string{
	"sdk-js":     "JavaScript SDK",
	"sdk-python": "Python SDK",
	"sdk-go":     "Go SDK",
	"sdk-rust":   "Rust SDK",
	"api-types":  "API Types",
}

func PackageLabel(name string) string {
	if label, ok := packageLabels[name]; ok {
		return label
	}

	if lang, ok := strings.CutPrefix(name, "sdk-"); ok && lang != "" {
		return strings.ToUpper(lang[:1]) + lang[1:] + " SDK"
	}

	return name
}
