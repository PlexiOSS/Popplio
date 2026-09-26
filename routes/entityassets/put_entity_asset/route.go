// Copyright (C) 2026 NodeByte LTD

// Package put_entity_asset implements PUT /{bots,servers,teams}/{id}/assets/{kind},
// which records the content version of an asset the frontend just uploaded to
// its CDN bucket. It is mounted once per entity router via Mount.
package put_entity_asset

import (
	"net/http"

	"popplio/api"
	"popplio/api/resp"
	"popplio/entityassets"
	"popplio/perms"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

// Mount registers the route for one entity type. urlPrefix is e.g. "/bots",
// idVar the chi URL param holding the entity ID, and perm the entity
// permission needed to change that entity's assets (the same one the
// frontend's upload gateway checks before writing the bytes).
func Mount(r *chi.Mux, targetType, urlPrefix, idVar string, perm perms.Perm) {
	uapi.Route{
		Pattern: urlPrefix + "/{" + idVar + "}/assets/{kind}",
		OpId:    "put_" + targetType + "_asset",
		Method:  uapi.PUT,
		Docs:    docsFor(targetType, idVar),
		Handler: routeFor(targetType, idVar),
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perm),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return targetType, chi.URLParam(r, idVar)
				},
			},
		},
	}.Route(r)
}

func docsFor(targetType, idVar string) func() *docs.Doc {
	return func() *docs.Doc {
		return &docs.Doc{
			Summary: "Put " + targetType + " asset version",
			Description: `Records the content version of a ` + targetType + ` asset (avatar/banner) that was just uploaded to the CDN, so it is returned in the entity's ` + "`asset_versions`" + `.

The file itself is uploaded through the frontend's upload gateway, not here. Returns a 204 on success.`,
			Params: []docs.Parameter{
				{
					Name:        idVar,
					Description: "The " + targetType + "'s ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
				{
					Name:        "kind",
					Description: "The asset kind: `banner`, or `avatar` for teams",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
			Req:  types.PutEntityAssetVersion{},
			Resp: types.ApiError{},
		}
	}
}

func routeFor(targetType, idVar string) func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
		targetID := chi.URLParam(r, idVar)
		kind := chi.URLParam(r, "kind")

		if !entityassets.KindAllowed(targetType, kind) {
			return resp.BadRequest("Unsupported asset kind for " + targetType + ": " + kind)
		}

		var payload types.PutEntityAssetVersion

		hresp, ok := uapi.MarshalReq(r, &payload)

		if !ok {
			return hresp
		}

		if !entityassets.ValidVersion(payload.Version) {
			return resp.BadRequest("version must be 16 to 64 lowercase hex characters")
		}

		if err := entityassets.Set(d.Context, targetType, targetID, kind, payload.Version); err != nil {
			return resp.Err("Error saving asset version", err, zap.String("target_type", targetType), zap.String("target_id", targetID), zap.String("kind", kind))
		}

		return resp.NoContent()
	}
}
