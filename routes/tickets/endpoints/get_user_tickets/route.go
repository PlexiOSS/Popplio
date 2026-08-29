package get_user_tickets

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
		Summary:     "Get User Tickets",
		Description: "Gets every ticket the given user has opened.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.TicketList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetUserTickets(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Failed to fetch tickets [db fetch]", err, zap.String("userId", d.Auth.ID))
	}

	ticketList := make([]types.Ticket, len(rows))
	for i, row := range rows {
		var messages []types.Message
		if err := json.Unmarshal(row.Messages, &messages); err != nil {
			return resp.Err("Failed to parse ticket messages [json]", err, zap.String("userId", d.Auth.ID))
		}

		var ticketContext map[string]string
		if err := json.Unmarshal(row.TicketContext, &ticketContext); err != nil {
			return resp.Err("Failed to parse ticket context [json]", err, zap.String("userId", d.Auth.ID))
		}

		ticketList[i] = types.Ticket{
			ID:            row.ID,
			ChannelID:     row.ChannelID,
			TopicID:       row.TopicID,
			Issue:         row.Issue,
			TicketContext: ticketContext,
			Messages:      messages,
			UserID:        row.UserID,
			CloseUserID:   row.CloseUserID,
			Open:          row.Open,
			CreatedAt:     row.CreatedAt.Time,
			EncKey:        row.EncKey,
		}
	}

	for i := range ticketList {
		author, err := dovewing.GetUser(d.Context, ticketList[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch ticket author [dovewing]", err, zap.String("userId", d.Auth.ID))
		}

		ticketList[i].Author = author
	}

	return uapi.HttpResponse{
		Json: types.TicketList{Tickets: ticketList},
	}
}
