// Copyright (C) 2026 NodeByte LTD

package emojis

import (
	"popplio/routes/emojis/endpoints/download_pack_emoji"
	"popplio/routes/emojis/endpoints/get_all_pack_emojis"
	"popplio/routes/emojis/endpoints/get_pack_emoji"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Emojis"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to pack emojis, addressed standalone"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/emojis/@all",
		OpId:    "get_all_pack_emojis",
		Method:  uapi.GET,
		Docs:    get_all_pack_emojis.Docs,
		Handler: get_all_pack_emojis.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/emojis/{id}",
		OpId:    "get_pack_emoji",
		Method:  uapi.GET,
		Docs:    get_pack_emoji.Docs,
		Handler: get_pack_emoji.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/emojis/{id}/download",
		OpId:    "download_pack_emoji",
		Method:  uapi.POST,
		Docs:    download_pack_emoji.Docs,
		Handler: download_pack_emoji.Route,
	}.Route(r)
}
