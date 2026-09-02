// Copyright (C) 2026 NodeByte LTD

package set_template_reaction

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
		Summary:     "Set Template Reaction",
		Description: "Sets, switches, or clears the caller's like/dislike on a server template. Sending the same reaction that's already active clears it. Returns the updated counts. Returns 204 on success",
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
			{
				Name:        "liked",
				Description: "Whether to like (true) or dislike (false) the template. Must be either true or false",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	uid := chi.URLParam(r, "uid")
	id := chi.URLParam(r, "id")

	likedStr := r.URL.Query().Get("liked")

	if likedStr != "true" && likedStr != "false" {
		return resp.BadRequest("liked must be either `true` or `false`")
	}

	liked := likedStr == "true"

	q := db.New(state.Pool)

	exists, err := q.CountServerTemplateByID(d.Context, id)

	if err != nil {
		return resp.Err("Error checking if template exists [db fetch]", err)
	}

	if !exists {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	existing, err := q.GetUserServerTemplateReaction(d.Context, db.GetUserServerTemplateReactionParams{
		TemplateID: id,
		UserID:     uid,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err := q.UpsertServerTemplateReaction(d.Context, db.UpsertServerTemplateReactionParams{
			TemplateID: id,
			UserID:     uid,
			Liked:      liked,
		}); err != nil {
			return resp.Err("Error while setting template reaction [db exec]", err)
		}
	case err != nil:
		return resp.Err("Error while checking existing template reaction [db fetch]", err)
	case existing == liked:
		if err := q.DeleteServerTemplateReaction(d.Context, db.DeleteServerTemplateReactionParams{
			TemplateID: id,
			UserID:     uid,
		}); err != nil {
			return resp.Err("Error while clearing template reaction [db exec]", err)
		}
	default:
		if err := q.UpsertServerTemplateReaction(d.Context, db.UpsertServerTemplateReactionParams{
			TemplateID: id,
			UserID:     uid,
			Liked:      liked,
		}); err != nil {
			return resp.Err("Error while switching template reaction [db exec]", err)
		}
	}

	counts, err := q.GetServerTemplateReactionCounts(d.Context, id)

	if err != nil {
		return resp.Err("Error while getting updated template reaction counts [db fetch]", err)
	}

	summary := types.TemplateReactionSummary{
		Likes:    int(counts.Likes),
		Dislikes: int(counts.Dislikes),
	}

	afterLiked, err := q.GetUserServerTemplateReaction(d.Context, db.GetUserServerTemplateReactionParams{
		TemplateID: id,
		UserID:     uid,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return resp.Err("Error while getting user's updated template reaction [db fetch]", err)
	default:
		summary.UserLiked = &afterLiked
	}

	return uapi.HttpResponse{
		Json: summary,
	}
}
