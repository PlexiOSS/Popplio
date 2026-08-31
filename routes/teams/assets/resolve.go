package assets

import "popplio/types"

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
