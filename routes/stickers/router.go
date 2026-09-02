// Copyright (C) 2026 NodeByte LTD

package stickers

import (
	"popplio/routes/stickers/endpoints/download_pack_sticker"
	"popplio/routes/stickers/endpoints/get_all_pack_stickers"
	"popplio/routes/stickers/endpoints/get_pack_sticker"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Stickers"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to pack stickers, addressed standalone"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/stickers/@all",
		OpId:    "get_all_pack_stickers",
		Method:  uapi.GET,
		Docs:    get_all_pack_stickers.Docs,
		Handler: get_all_pack_stickers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/stickers/{id}",
		OpId:    "get_pack_sticker",
		Method:  uapi.GET,
		Docs:    get_pack_sticker.Docs,
		Handler: get_pack_sticker.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/stickers/{id}/download",
		OpId:    "download_pack_sticker",
		Method:  uapi.POST,
		Docs:    download_pack_sticker.Docs,
		Handler: download_pack_sticker.Route,
	}.Route(r)
}
