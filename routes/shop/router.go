// Package shop mounts the "Shop" group of API routes.
//
// These API endpoints are related to the IBL shop.
package shop

import (
	"net/http"

	"popplio/api"
	"popplio/perms"
	"popplio/routes/shop/endpoints/get_public_coupons"
	"popplio/routes/shop/endpoints/get_shop_item_benefits"
	"popplio/routes/shop/endpoints/get_shop_items"
	"popplio/routes/shop/endpoints/get_shop_purchases"
	"popplio/routes/shop/endpoints/purchase_shop_item"
	"popplio/validators"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Shop"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to the IBL shop."
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/shop/public-coupons",
		OpId:    "get_public_coupons",
		Method:  uapi.GET,
		Docs:    get_public_coupons.Docs,
		Handler: get_public_coupons.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/shop/items",
		OpId:    "get_shop_items",
		Method:  uapi.GET,
		Docs:    get_shop_items.Docs,
		Handler: get_shop_items.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/shop/item-benefits",
		OpId:    "get_shop_item_benefits",
		Method:  uapi.GET,
		Docs:    get_shop_item_benefits.Docs,
		Handler: get_shop_item_benefits.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/shop/purchases",
		OpId:    "get_shop_purchases",
		Method:  uapi.GET,
		Docs:    get_shop_purchases.Docs,
		Handler: get_shop_purchases.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/{target_type}/{target_id}/shop/purchase",
		OpId:    "purchase_shop_item",
		Method:  uapi.POST,
		Docs:    purchase_shop_item.Docs,
		Handler: purchase_shop_item.Route,
		Auth:    api.GetAllAuthTypes(),
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perms.EntityBuyShopItems),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return validators.NormalizeTargetType(chi.URLParam(r, "target_type")), chi.URLParam(r, "target_id")
				},
			},
		},
	}.Route(r)
}
