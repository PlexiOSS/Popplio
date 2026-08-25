// Package get_vote_captcha_challenge implements GET /votes/captcha/challenge
// — "Get Vote Captcha Challenge".
//
// Issues a fresh, signed proof-of-work challenge that must be solved and
// submitted alongside a vote (see create_user_entity_vote) when the target
// entity requires it. No authentication or server-side state is needed to
// issue a challenge; see the popplio/captcha package for the full protocol.
package get_vote_captcha_challenge

import (
	"net/http"

	"popplio/api/resp"
	"popplio/captcha"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Vote Captcha Challenge",
		Description: "Issues a proof-of-work captcha challenge that must be solved and submitted alongside a vote",
		Resp:        captcha.Challenge{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	ch, err := captcha.New()

	if err != nil {
		return resp.ErrBody("Failed to issue captcha challenge", "Failed to issue captcha challenge.", err)
	}

	return uapi.HttpResponse{
		Json: ch,
	}
}
