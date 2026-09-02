// Copyright (C) 2026 NodeByte LTD

package download_pack_emoji

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

type DownloadResponse struct {
	Downloads int `json:"downloads"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Download Pack Emoji",
		Description: "Records a single-emoji download and returns the new total. Returns 204 on success",
		Resp:        DownloadResponse{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The emoji's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	downloads, err := db.New(state.Pool).IncrementPackEmojiDownloads(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while incrementing emoji downloads [db exec]", err)
	}

	return uapi.HttpResponse{
		Json: DownloadResponse{Downloads: int(downloads)},
	}
}
