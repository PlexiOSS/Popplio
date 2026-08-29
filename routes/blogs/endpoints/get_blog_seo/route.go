// Package get_blog_seo implements GET /blogs/{slug}/seo — "Get Blog Post".
//
// Gets the minimal SEO information about a blogpost for embed/search
// purposes. Used by v4 website for meta tags
package get_blog_seo

import (
	"net/http"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog Post",
		Description: "Gets the minimal SEO information about a blogpost for embed/search purposes. Used by v4 website for meta tags",
		Resp:        types.SEO{},
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
	slug := chi.URLParam(r, "slug")

	row, err := db.New(state.Pool).GetBlogSEO(d.Context, slug)

	if err != nil {
		state.Logger.Error("Error fetching blog post [db query]", zap.Error(err), zap.String("slug", slug))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	seo := types.SEO{
		ID:     slug,
		Name:   row.Title,
		Avatar: "",
		Short:  row.Description,
	}

	return uapi.HttpResponse{
		Json: seo,
	}
}
