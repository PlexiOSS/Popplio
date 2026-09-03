// Copyright (C) 2026 NodeByte LTD

package get_pack_emoji

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
		Summary:     "Get Pack Emoji",
		Description: "Gets a single pack emoji by its own ID, along with its owning pack's identity and owner",
		Resp:        types.PackEmojiDetail{},
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

	q := db.New(state.Pool)

	row, err := q.GetPackEmojiByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting pack emoji [db fetch]", err)
	}

	pack, err := q.GetPackByURL(d.Context, row.PackUrl)

	if errors.Is(err, pgx.ErrNoRows) {
		// The pack was deleted out from under an emoji that somehow
		// survived (shouldn't happen -- pack_emojis cascades on pack
		// delete -- but favor a clean 404 over a broken detail page).
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
		Json: types.PackEmojiDetail{
			ID:        row.ID,
			Name:      row.Name,
			Animated:  row.Animated,
			Position:  int(row.Position),
			Downloads: int(row.Downloads),
			CreatedAt: row.CreatedAt.Time,
			PackURL:   row.PackUrl,
			PackName:  pack.Name,
			Owner:     owner,
			Vanity:    vanityCode,
		},
	}
}
