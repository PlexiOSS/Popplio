// Package get_blog_list implements GET /blogs/@all — "Get Blog List".
//
// Gets all blog posts on the list in condensed form
package get_blog_list

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Blog List",
		Description: "Gets all blog posts on the list in condensed form",
		Resp:        types.Blog{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetBlogList(d.Context)

	if err != nil {
		return resp.Err("Error while fetching blog posts [db query]", err)
	}

	blogPosts := make([]types.BlogListPost, len(rows))
	for i, row := range rows {
		blogPosts[i] = types.BlogListPost{
			Slug:        row.Slug,
			Title:       row.Title,
			Description: row.Description,
			UserID:      row.UserID,
			CreatedAt:   row.CreatedAt.Time,
			Draft:       row.Draft,
			Tags:        row.Tags,
		}
	}

	for i := range blogPosts {
		blogPosts[i].Author, err = dovewing.GetUser(d.Context, blogPosts[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error while getting user [dovewing]", err, zap.String("user_id", blogPosts[i].UserID))
		}
	}

	return uapi.HttpResponse{
		Json: types.Blog{
			Posts: blogPosts,
		},
	}
}
