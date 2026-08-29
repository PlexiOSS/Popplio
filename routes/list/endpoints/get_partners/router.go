// Package get_partners implements GET /list/partners — "Get List Partners".
//
// Gets the official partners of the list
package get_partners

import (
	"encoding/json"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Partners",
		Description: "Gets the official partners of the list",
		Resp:        types.PartnerList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	q := db.New(state.Pool)

	partnerRows, err := q.GetPartners(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch partner list [db fetch]", err)
	}

	partners := make([]types.Partner, len(partnerRows))
	for i, row := range partnerRows {
		var links []types.Link
		if err := json.Unmarshal(row.Links, &links); err != nil {
			return resp.ErrBody("Failed to parse partner links", "Could not parse links for partner "+row.ID+".", err, zap.String("partner_id", row.ID))
		}

		var botID *string
		if row.BotID.Valid {
			botID = &row.BotID.String
		}

		partners[i] = types.Partner{
			ID:        row.ID,
			Name:      row.Name,
			Short:     row.Short,
			Links:     links,
			Type:      row.Type,
			CreatedAt: row.CreatedAt.Time,
			UserID:    row.UserID,
			BotID:     botID,
		}
	}

	for i := range partners {
		err := state.Validator.Struct(partners[i])

		if err != nil {
			return resp.ErrBody("Failed to validate partner", "Could not validate partner "+partners[i].ID+".", err, zap.String("partner_id", partners[i].ID))
		}

		partners[i].User, err = dovewing.GetUser(d.Context, partners[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch partner user", err, zap.String("partner_id", partners[i].ID), zap.String("user_id", partners[i].UserID))
		}

	}

	typeRows, err := q.GetPartnerTypes(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch partner types [db fetch]", err)
	}

	partnerTypes := make([]types.PartnerTypes, len(typeRows))
	for i, row := range typeRows {
		partnerTypes[i] = types.PartnerTypes{
			ID:        row.ID,
			Name:      row.Name,
			Short:     row.Short,
			Icon:      row.Icon,
			CreatedAt: row.CreatedAt.Time,
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.PartnerList{
			Partners:     partners,
			PartnerTypes: partnerTypes,
		},
	}
}
