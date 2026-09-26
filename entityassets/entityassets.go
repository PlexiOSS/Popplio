// Copyright (C) 2026 NodeByte LTD

package entityassets

import (
	"context"
	"regexp"
	"slices"

	"popplio/db"
	"popplio/state"
)

const (
	KindAvatar = "avatar"
	KindBanner = "banner"
)

var allowedKinds = map[string][]string{
	"bot":    {KindBanner},
	"server": {KindBanner},
	"team":   {KindAvatar, KindBanner},
}

var versionRe = regexp.MustCompile(`^[0-9a-f]{16,64}$`)

func KindAllowed(targetType, kind string) bool {
	return slices.Contains(allowedKinds[targetType], kind)
}

func ValidVersion(version string) bool {
	return versionRe.MatchString(version)
}

func Set(ctx context.Context, targetType, targetID, kind, version string) error {
	return db.New(state.Pool).UpsertEntityAssetVersion(ctx, db.UpsertEntityAssetVersionParams{
		TargetType: targetType,
		TargetID:   targetID,
		Kind:       kind,
		Version:    version,
	})
}

func GetMany(ctx context.Context, targetType string, targetIDs []string) (map[string]map[string]string, error) {
	out := make(map[string]map[string]string, len(targetIDs))

	for _, id := range targetIDs {
		out[id] = map[string]string{}
	}

	if len(targetIDs) == 0 {
		return out, nil
	}

	rows, err := db.New(state.Pool).GetEntityAssetVersions(ctx, db.GetEntityAssetVersionsParams{
		TargetType: targetType,
		TargetIds:  targetIDs,
	})

	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if versions, ok := out[row.TargetID]; ok {
			versions[row.Kind] = row.Version
		}
	}

	return out, nil
}

func Get(ctx context.Context, targetType, targetID string) (map[string]string, error) {
	versions, err := GetMany(ctx, targetType, []string{targetID})

	if err != nil {
		return nil, err
	}

	return versions[targetID], nil
}
