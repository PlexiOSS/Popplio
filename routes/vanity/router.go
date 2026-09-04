// Package vanity mounts the "Vanity" group of API routes.
//
// These API endpoints are related to vanity codes on IGL
package vanity

import (
	"net/http"

	"popplio/api"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/vanity/endpoints/patch_vanity"
	"popplio/routes/vanity/endpoints/redirect_vanity"
	"popplio/routes/vanity/endpoints/resolve_vanity"
	"popplio/state"
	"popplio/validators"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Vanity"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to vanity codes on IGL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/vanity/{code}",
		OpId:    "resolve_vanity",
		Method:  uapi.GET,
		Docs:    resolve_vanity.Docs,
		Handler: resolve_vanity.Route,
	}.Route(r)

	uapi.Route{
		Pattern:               "/@{code}",
		OpId:                  "redirect_vanity",
		Method:                uapi.GET,
		Docs:                  redirect_vanity.Docs,
		Handler:               redirect_vanity.Route,
		DisablePathSlashCheck: true,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/vanity",
		OpId:    "patch_vanity",
		Method:  uapi.PATCH,
		Docs:    patch_vanity.Docs,
		Handler: patch_vanity.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perms.EntitySetVanity),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
					targetId := chi.URLParam(r, "target_id")

					// pack_emoji/pack_sticker aren't real auth entities (no
					// AuthTypeMap entry, no team permissions) -- ownership is
					// owner-only on the parent pack, same model packs/themes
					// already use. Resolve down to the owning user instead so
					// AuthzEntityPermissionCheck's normal "user" handling
					// (self-match, or ErrUsersCannotModifyOtherUsers) applies.
					q := db.New(state.Pool)
					switch targetType {
					case "pack_emoji":
						emoji, err := q.GetPackEmojiByID(r.Context(), targetId)
						if err != nil {
							return "", ""
						}
						owner, err := q.GetPackOwner(r.Context(), emoji.PackUrl)
						if err != nil {
							return "", ""
						}
						return api.TargetTypeUser, owner
					case "pack_sticker":
						sticker, err := q.GetPackStickerByID(r.Context(), targetId)
						if err != nil {
							return "", ""
						}
						owner, err := q.GetPackOwner(r.Context(), sticker.PackUrl)
						if err != nil {
							return "", ""
						}
						return api.TargetTypeUser, owner
					case "pack_sound":
						sound, err := q.GetPackSoundByID(r.Context(), targetId)
						if err != nil {
							return "", ""
						}
						owner, err := q.GetPackOwner(r.Context(), sound.PackUrl)
						if err != nil {
							return "", ""
						}
						return api.TargetTypeUser, owner
					default:
						return targetType, targetId
					}
				},
			},
		},
	}.Route(r)
}
