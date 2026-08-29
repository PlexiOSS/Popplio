// Package get_user_reminders implements GET /users/{id}/reminders — "Get
// User Reminders".
//
// Gets a users reminders
package get_user_reminders

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Reminders",
		Description: "Gets a users reminders",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ReminderList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	q := db.New(state.Pool)

	// Fetch reminder from postgres
	rows, err := q.GetUserReminders(d.Context, id)

	if err != nil {
		return resp.Err("Error querying reminders [db fetch]", err, zap.String("user_id", id))
	}

	reminders := make([]types.Reminder, len(rows))
	for i, row := range rows {
		reminders[i] = types.Reminder{
			UserID:     row.UserID,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			CreatedAt:  row.CreatedAt.Time,
			LastAcked:  row.LastAcked.Time,
		}
	}

	for i, reminder := range reminders {
		// Try resolving the entity from discord API
		reminders[i].Resolved = &types.ResolvedReminder{
			Name:   "Unknown",
			Avatar: "https://cdn.discordapp.com/embed/avatars/0.png",
		}

		switch reminder.TargetType {
		case "bot":
			bot, err := dovewing.GetUser(d.Context, reminder.TargetID, state.DovewingPlatformDiscord)

			if err == nil {
				reminders[i].Resolved = &types.ResolvedReminder{
					Name:   bot.Username,
					Avatar: bot.Avatar,
				}
			}
		case "server":
			row, err := q.GetServerNameAndAvatar(d.Context, reminder.TargetID)

			if err == nil {
				reminders[i].Resolved = &types.ResolvedReminder{
					Name:   row.Name,
					Avatar: row.Avatar,
				}
			}
		case "team":
			name, err := q.GetTeamName(d.Context, reminder.TargetID)

			if err == nil {
				reminders[i].Resolved = &types.ResolvedReminder{
					Name: name,
				}
			}
		}
	}

	reminderList := types.ReminderList{
		Reminders: reminders,
	}

	return uapi.HttpResponse{
		Json: reminderList,
	}
}
