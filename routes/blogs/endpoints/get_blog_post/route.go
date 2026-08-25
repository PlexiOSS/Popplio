// Package get_blog_post implements GET /blogs/{slug} — "Get Blog Post".
//
// Gets a blog posts on the list
package get_blog_post

import (
	"errors"
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

var (
	blogColsArr = dbutil.GetCols(types.BlogPost{})

	blogCols = strings.Join(blogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog Post",
		Description: "Gets a blog posts on the list",
		Resp:        types.BlogPost{},
		Params: []docs.Parameter{
			{
				Name:        "slug",
				Description: "The slug of the blog post",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	row, err := state.Pool.Query(d.Context, "SELECT "+blogCols+" FROM blogs WHERE slug = $1", chi.URLParam(r, "slug"))

	if err != nil {
		return resp.Err("Error fetching blog post [db query]", err, zap.String("slug", chi.URLParam(r, "slug")))
	}

	blogPost, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.BlogPost])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error fetching blog post [collect]", err, zap.String("slug", chi.URLParam(r, "slug")))
	}

	blogPost.Author, err = dovewing.GetUser(d.Context, blogPost.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error while getting user [dovewing]", err, zap.String("user_id", blogPost.UserID))
	}

	return uapi.HttpResponse{
		Json: blogPost,
	}
}
