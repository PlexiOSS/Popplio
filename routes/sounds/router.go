// Copyright (C) 2026 NodeByte LTD

package sounds

import (
	"popplio/routes/sounds/endpoints/download_pack_sound"
	"popplio/routes/sounds/endpoints/get_all_pack_sounds"
	"popplio/routes/sounds/endpoints/get_pack_sound"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Sounds"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to pack sounds, addressed standalone"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/sounds/@all",
		OpId:    "get_all_pack_sounds",
		Method:  uapi.GET,
		Docs:    get_all_pack_sounds.Docs,
		Handler: get_all_pack_sounds.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/sounds/{id}",
		OpId:    "get_pack_sound",
		Method:  uapi.GET,
		Docs:    get_pack_sound.Docs,
		Handler: get_pack_sound.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/sounds/{id}/download",
		OpId:    "download_pack_sound",
		Method:  uapi.POST,
		Docs:    download_pack_sound.Docs,
		Handler: download_pack_sound.Route,
	}.Route(r)
}
