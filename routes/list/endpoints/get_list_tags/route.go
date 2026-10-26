// Package get_list_tags implements GET /list/tags — "Get List Tags".
//
// Every tag used by a listed bot or public listed server, with how many of
// each use it.
package get_list_tags

import (
	"encoding/json"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const cacheKey = "list-tags"

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Tags",
		Description: "Returns every tag used by an approved or certified bot or a public, approved or certified server, matched case-insensitively, with how many bots and servers use it. Most used first. Cached for 30 minutes.",
		Resp:        types.ListTags{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	if cached, err := state.Redis.Get(d.Context, cacheKey).Bytes(); err == nil {
		var out types.ListTags

		if json.Unmarshal(cached, &out) == nil {
			return uapi.HttpResponse{Json: out}
		}
	}

	rows, err := db.New(state.Pool).GetListTags(d.Context)

	if err != nil {
		return resp.Err("Failed to fetch tags", err)
	}

	out := types.ListTags{Tags: make([]types.ListTag, 0, len(rows))}

	for _, row := range rows {
		out.Tags = append(out.Tags, types.ListTag{Tag: row.Label, Bots: row.Bots, Servers: row.Servers})
	}

	if b, err := json.Marshal(out); err == nil {
		state.Redis.Set(d.Context, cacheKey, b, 30*time.Minute)
	}

	return uapi.HttpResponse{Json: out}
}
