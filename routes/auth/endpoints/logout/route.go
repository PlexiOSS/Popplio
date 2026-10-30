package logout

import (
	"net/http"

	"popplio/api/resp"
	"popplio/api/sessioncookie"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Log Out",
		Description: "Ends the session used to make this request, if any, and clears the website's session cookie. Always succeeds.",
		Resp:        types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if d.Auth.Authorized {
		if sessionID, ok := d.Auth.Data["session_id"].(string); ok && sessionID != "" {
			err := db.New(state.Pool).DeleteSession(d.Context, db.DeleteSessionParams{
				ID:         sessionID,
				TargetID:   d.Auth.ID,
				TargetType: d.Auth.TargetType,
			})

			if err != nil {
				return resp.Err("Failed to end session", err, zap.String("userID", d.Auth.ID))
			}
		}
	}

	return resp.WithHeaders(resp.NoContent(), map[string]string{
		"Set-Cookie": sessioncookie.Clear(),
	})
}
