// Copyright (C) 2026 NodeByte LTD

package get_pack_sound

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	packAssets "popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Pack Sound",
		Description: "Gets a single pack sound by its own ID, along with its owning pack's identity and owner",
		Resp:        types.PackSoundDetail{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The sound's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	q := db.New(state.Pool)

	row, err := q.GetPackSoundByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting pack sound [db fetch]", err)
	}

	pack, err := q.GetPackByURL(d.Context, row.PackUrl)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting owning pack [db fetch]", err)
	}

	owner, err := dovewing.GetUser(d.Context, pack.Owner, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error querying dovewing for owner user", err)
	}

	vanityCode, err := packAssets.ResolveVanityCode(d.Context, q, row.ID)

	if err != nil {
		return resp.Err("Error while resolving vanity", err)
	}

	return uapi.HttpResponse{
		Json: types.PackSoundDetail{
			ID:         row.ID,
			Name:       row.Name,
			DurationMs: int(row.DurationMs),
			Position:   int(row.Position),
			Downloads:  int(row.Downloads),
			CreatedAt:  row.CreatedAt.Time,
			PackURL:    row.PackUrl,
			PackName:   pack.Name,
			Owner:      owner,
			Vanity:     vanityCode,
		},
	}
}
