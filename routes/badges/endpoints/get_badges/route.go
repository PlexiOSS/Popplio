package get_badges

import (
	"net/http"

	"popplio/api/resp"
	"popplio/badgerules"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Badges",
		Description: "Lists every badge in the catalog with how to earn it and how many entities hold it.",
		Resp:        types.PublicBadgeList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).ListPublicBadges(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch badges [db fetch]", err)
	}

	out := types.PublicBadgeList{Badges: make([]types.PublicBadge, 0, len(rows))}

	for _, row := range rows {
		howTo, automatic := badgerules.Describe(row.AutoRule)

		if !automatic {
			howTo = "Awarded by Omniplex staff."
		}

		out.Badges = append(out.Badges, types.PublicBadge{
			BadgeCatalog: types.BadgeCatalog{
				ID:          row.ID,
				Name:        row.Name,
				Description: row.Description,
				Icon:        row.Icon,
				Color:       row.Color,
				TargetTypes: row.TargetTypes,
			},
			Automatic:  automatic,
			HowToEarn:  howTo,
			LinkedRole: row.LinkedRole,
			Holders:    int(row.Holders),
		})
	}

	return uapi.HttpResponse{Json: out}
}
