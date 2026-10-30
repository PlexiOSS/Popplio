// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"

	"popplio/changeloggen"
	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

var changelogProjects = []string{"popplio", "omniplex", "keel"}

func ChangelogDrafts(ctx context.Context) error {
	if len(state.Config.Arcadia.Owners) == 0 {
		return errors.New("no arcadia.owners configured to credit changelog drafts to")
	}

	author := state.Config.Arcadia.Owners[0].String()

	var errs []error

	for _, project := range changelogProjects {
		owner, repoName, ok := changeloggen.RepoFor(project)

		if !ok {
			continue
		}

		sources := []changelogSource{{paths: changeloggen.ChangelogPathsFor(project)}}

		if dir := changeloggen.PackagesDirFor(project); dir != "" {
			packages, err := changeloggen.ListPackages(ctx, owner, repoName, dir)

			if err != nil {
				errs = append(errs, fmt.Errorf("%s: listing packages: %w", project, err))
			}

			for _, pkg := range packages {
				sources = append(sources, changelogSource{pkg: pkg, paths: []string{dir + "/" + pkg + "/CHANGELOG.md"}})
			}
		}

		for _, source := range sources {
			created, err := draftNewReleases(ctx, project, owner, repoName, source, author)

			if err != nil {
				errs = append(errs, fmt.Errorf("%s %s: %w", project, source.pkg, err))
				continue
			}

			for _, version := range created {
				state.Logger.Info("changelog_drafts: created unpublished draft",
					zap.String("project", project), zap.String("package", source.pkg), zap.String("version", version))
			}
		}
	}

	return errors.Join(errs...)
}

type changelogSource struct {
	pkg   string
	paths []string
}

func draftNewReleases(ctx context.Context, project, owner, repoName string, source changelogSource, author string) ([]string, error) {
	markdown, err := changeloggen.FetchChangelogFile(ctx, owner, repoName, source.paths)

	if errors.Is(err, changeloggen.ErrNoChangelog) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	releases := changeloggen.ParseReleases(markdown)

	if len(releases) == 0 {
		return nil, nil
	}

	q := db.New(state.Pool)

	var pkg pgtype.Text

	if source.pkg != "" {
		pkg = pgtype.Text{String: source.pkg, Valid: true}
	}

	existing, err := q.GetChangelogVersions(ctx, db.GetChangelogVersionsParams{Project: project, Package: pkg})

	if err != nil {
		return nil, fmt.Errorf("reading existing versions: %w", err)
	}

	known := make(map[string]bool, len(existing))
	latest := ""

	for _, v := range existing {
		known[v] = true

		if latest == "" || changeloggen.CompareVersions(v, latest) > 0 {
			latest = v
		}
	}

	var due []changeloggen.Release

	for _, r := range releases {
		if known[r.Version] {
			continue
		}

		if latest == "" {
			if len(due) == 0 || changeloggen.CompareVersions(r.Version, due[0].Version) > 0 {
				due = []changeloggen.Release{r}
			}
			continue
		}

		if changeloggen.CompareVersions(r.Version, latest) > 0 {
			due = append(due, r)
		}
	}

	var created []string

	for _, r := range due {
		draft, err := changeloggen.SummarizeRelease(ctx, project, source.pkg, r)

		if err != nil {
			return created, fmt.Errorf("summarizing %s: %w", r.Version, err)
		}

		err = q.InsertChangelogEntry(ctx, db.InsertChangelogEntryParams{
			Project:          project,
			Version:          r.Version,
			Added:            nonNilStrings(draft.Added),
			Updated:          nonNilStrings(draft.Updated),
			Fixed:            nonNilStrings(draft.Fixed),
			Security:         nonNilStrings(draft.Security),
			Removed:          nonNilStrings(draft.Removed),
			ExtraDescription: draft.ExtraDescription,
			Prerelease:       false,
			Published:        false,
			CreatedBy:        author,
			Package:          pkg,
		})

		if err != nil {
			return created, fmt.Errorf("inserting draft %s: %w", r.Version, err)
		}

		created = append(created, r.Version)
	}

	return created, nil
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}

	return s
}
