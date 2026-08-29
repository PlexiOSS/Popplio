package get_public_team

import (
	"net/http"
	"sort"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Public Team",
		Description: "Returns the public staff roster: who's on staff and what position(s) they hold.",
		Resp:        []types.PublicStaffMember{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	q := db.New(state.Pool)

	members, err := q.GetStaffMembersRoster(d.Context)

	if err != nil {
		return resp.Err("Error while fetching staff members [db fetch]", err)
	}

	positionList, err := q.GetStaffPositionsRoster(d.Context)

	if err != nil {
		return resp.Err("Error while fetching staff positions [db fetch]", err)
	}

	positionsByID := make(map[pgtype.UUID]db.GetStaffPositionsRosterRow, len(positionList))
	for _, p := range positionList {
		positionsByID[p.ID] = p
	}

	team := make([]types.PublicStaffMember, 0, len(members))

	for _, member := range members {
		user, err := dovewing.GetUser(d.Context, member.UserID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Warn("Failed to resolve staff member for public team roster", zap.Error(err), zap.String("userID", member.UserID))
			continue
		}

		if user.Bot {
			continue
		}

		positions := make([]types.PublicStaffPosition, 0, len(member.Positions))
		for _, id := range member.Positions {
			p, ok := positionsByID[id]
			if !ok {
				continue
			}
			positions = append(positions, types.PublicStaffPosition{
				Name:  p.Name,
				Icon:  p.Icon,
				Index: p.Index,
			})
		}

		sort.Slice(positions, func(i, j int) bool { return positions[i].Index < positions[j].Index })

		team = append(team, types.PublicStaffMember{
			UserID:      member.UserID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Avatar:      user.Avatar,
			Positions:   positions,
		})
	}

	sort.Slice(team, func(i, j int) bool {
		iRank, jRank := int32(1<<31-1), int32(1<<31-1)
		if len(team[i].Positions) > 0 {
			iRank = team[i].Positions[0].Index
		}
		if len(team[j].Positions) > 0 {
			jRank = team[j].Positions[0].Index
		}
		if iRank != jRank {
			return iRank < jRank
		}
		return team[i].Username < team[j].Username
	})

	return uapi.HttpResponse{
		Json: team,
	}
}
