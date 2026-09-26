package assets

import (
	"context"

	"popplio/api"
	"popplio/entityassets"
	"popplio/types"
)

func ResolveIndexTeam(team *types.Team) {
	if team.Tags == nil {
		team.Tags = []string{}
	}

	if team.ExtraLinks == nil {
		team.ExtraLinks = []types.Link{}
	}

	team.Votes = team.ApproximateVotes
}

func ResolveIndexTeams(teams []types.Team) {
	for i := range teams {
		ResolveIndexTeam(&teams[i])
	}
}

// ResolveTeamAssetVersions fills in AssetVersions for every team in one query.
func ResolveTeamAssetVersions(ctx context.Context, teams []types.Team) error {
	ids := make([]string, len(teams))
	for i := range teams {
		ids[i] = teams[i].ID
	}

	versions, err := entityassets.GetMany(ctx, api.TargetTypeTeam, ids)
	if err != nil {
		return err
	}

	for i := range teams {
		teams[i].AssetVersions = versions[teams[i].ID]
	}

	return nil
}
