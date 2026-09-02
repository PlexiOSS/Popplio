// Package redirect_vanity implements GET /@{code} — "Redirect Vanity".
//
// Resolve a vanity by its code or (then) its target id, then redirects to
// the API url. Useful for debugging mainly
package redirect_vanity

import (
	"net/http"

	"popplio/api/resp"

	"popplio/routes/vanity/assets"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Redirect Vanity",
		Description: "Resolve a vanity by its code or (then) its target id, then redirects to the API url. Useful for debugging mainly",
		Resp:        types.Vanity{},
		Params: []docs.Parameter{
			{
				Name:        "code",
				In:          "path",
				Description: "The vanity code",
				Required:    true,
				Schema:      docs.IdSchema,
			},
			{
				Name:        "itag",
				In:          "query",
				Description: "Resolve based on itag",
				Required:    false,
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Fetch the vanity, first attempt code
	code := chi.URLParam(r, "code")
	itag := r.URL.Query().Get("itag")

	var v *types.Vanity
	var err error

	if itag == "true" {
		v, err = assets.ResolveVanityByItag(d.Context, code)
	} else {
		v, err = assets.ResolveVanity(d.Context, code)
	}

	if err != nil {
		return resp.ErrBody("Failed to resolve vanity", "An internal error occurred", err, zap.String("code", code))
	}

	if v == nil {
		return resp.NotFound("This entity does not exist")
	}

	return uapi.HttpResponse{
		Redirect: redirectPath(v.TargetType) + "/" + v.TargetID,
	}
}

// redirectPath maps a vanity's target_type to the frontend path it lives
// under. Naive "+s" pluralization covers bot/server/team/pack; pack_emoji
// and pack_sticker need a real mapping since they don't live under a
// "pack_emojis"/"pack_stickers" path at all.
func redirectPath(targetType string) string {
	switch targetType {
	case "pack_emoji":
		return "/emojis"
	case "pack_sticker":
		return "/stickers"
	default:
		return "/" + targetType + "s"
	}
}
