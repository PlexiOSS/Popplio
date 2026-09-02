// Copyright (C) 2026 NodeByte LTD

package get_template_reaction

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Template Reaction",
		Description: "Gets a server template's like/dislike counts, plus the given user's own reaction if any. No session required -- reactions aren't private.",
		Resp:        types.TemplateReactionSummary{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "id",
				Description: "The template's internal ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "uid")
	id := chi.URLParam(r, "id")

	q := db.New(state.Pool)

	counts, err := q.GetServerTemplateReactionCounts(d.Context, id)

	if err != nil {
		return resp.Err("Error while getting template reaction counts [db fetch]", err)
	}

	summary := types.TemplateReactionSummary{
		Likes:    int(counts.Likes),
		Dislikes: int(counts.Dislikes),
	}

	liked, err := q.GetUserServerTemplateReaction(d.Context, db.GetUserServerTemplateReactionParams{
		TemplateID: id,
		UserID:     uid,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return resp.Err("Error while getting user's template reaction [db fetch]", err)
	default:
		summary.UserLiked = &liked
	}

	return uapi.HttpResponse{
		Json: summary,
	}
}
