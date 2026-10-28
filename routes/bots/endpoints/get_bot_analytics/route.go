package get_bot_analytics

import (
	"net/http"
	"strconv"

	"popplio/analytics"
	"popplio/api/resp"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot Analytics",
		Description: "Returns the bot's daily votes, views, invite clicks and counts over time. You must have 'Edit Bots' in the bot's team.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The bot's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "days",
				Description: "How many days to return, ending today (UTC). Between 7 and 365, defaults to 30.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.EntityAnalytics{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	days := analytics.DefaultDays
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "days must be a number"},
			}
		}

		days = n
	}

	out, err := analytics.Get(d.Context, "bot", id, days)

	if err != nil {
		return resp.Err("Failed to load bot analytics", err, zap.String("botID", id))
	}

	return uapi.HttpResponse{Json: out}
}
