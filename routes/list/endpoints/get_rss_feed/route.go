// Package get_rss_feed implements GET /list/rss.xml — "Get RSS Feed".
//
// Gets the RSS feed for the site, in XML format
package get_rss_feed

import (
	"net/http"
	"strconv"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/seo"
	"popplio/seo/fetchers"
	"popplio/state"

	"encoding/xml"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get RSS Feed",
		Description: "Gets the RSS feed for the site, in XML format",
		Resp:        seo.RssFeed{},
	}
}

const perPage = 10

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	// Check cache, this is how we can avoid hefty ratelimits
	cache := state.Redis.Get(d.Context, "rssfeed-"+strconv.FormatUint(pageNum, 10)).Val()
	if cache != "" {
		return uapi.HttpResponse{
			Data: cache,
			Headers: map[string]string{
				"X-Popplio-Cached": "true",
				"Content-Type":     "application/xml",
			},
		}
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	rssFeed := seo.RssFeed{}

	rssFeed.NS = "http://www.w3.org/2005/Atom"
	rssFeed.Version = "2.0"
	rssFeed.Channel = &seo.RssChannel{
		Title:         "Omniplex",
		Link:          state.Config.Sites.Frontend,
		Description:   "Search our vast list of bots for an exciting start to your server.",
		Language:      "en-us",
		LastBuildDate: time.Now().Format(time.RFC822),
		Copyright:     "Copyright " + time.Now().Format("2006") + " NodeByte LTD",
		Docs:          "https://www.rssboard.org/rss-specification",
		TTL:           120,
		Category:      []string{"Bots", "Servers"},
		AtomLink: []*seo.RssAtomLink{
			{
				Href: func() string {
					d := state.Config.Sites.API + r.URL.Path

					if r.URL.RawQuery != "" {
						d += "?" + r.URL.RawQuery
					}

					return d
				}(),
				Rel:  "self",
				Type: "application/rss+xml",
			},
			{
				Href: state.Config.Sites.API + "/list/rss.xml",
				Rel:  "first",
				Type: "application/rss+xml",
			},
			{
				Href: state.Config.Sites.API + "/list/rss.xml?page=" + strconv.FormatUint(pageNum+1, 10),
				Rel:  "next",
				Type: "application/rss+xml",
			},
		},
		Links: []*seo.RssLink{
			{
				Href: state.Config.Sites.API + "/list/rss.xml",
				Rel:  "first",
			},
			{
				Href: state.Config.Sites.API + "/list/rss.xml?page=" + strconv.FormatUint(pageNum+1, 10),
				Rel:  "next",
			},
		},
		Generator: "Popplio RSS Generator",
	}

	if pageNum > 1 {
		rssFeed.Channel.AtomLink = append(rssFeed.Channel.AtomLink, &seo.RssAtomLink{
			Href: state.Config.Sites.API + "/list/rss.xml?page=" + strconv.FormatUint(pageNum-1, 10),
			Rel:  "prev",
			Type: "application/rss+xml",
		})
		rssFeed.Channel.Links = append(rssFeed.Channel.Links, &seo.RssLink{
			Href: state.Config.Sites.API + "/list/rss.xml?page=" + strconv.FormatUint(pageNum-1, 10),
			Rel:  "prev",
		})

	}

	var collector = seo.IDCollector{}
	q := db.New(state.Pool)

	// Get new bots
	newBotIDs, err := q.GetNewBotIDs(d.Context, db.GetNewBotIDsParams{Limit: int32(limit), Offset: int32(offset)})

	if err != nil {
		return resp.Err("Failed to get bots [row query] for generating RSS feed", err)
	}

	newBots := collector.CollectIDs(newBotIDs)

	for _, id := range newBots {
		err := state.SeoMapGenerator.AddToRss(d.Context, &fetchers.BotFetcher{}, &rssFeed, "New Bots", id)

		if err != nil {
			return resp.Err("Failed to add bot to RSS feed", err, zap.String("botId", id))
		}
	}

	// Get certified bots
	certBotIDs, err := q.GetCertifiedBotIDs(d.Context, db.GetCertifiedBotIDsParams{Limit: int32(limit), Offset: int32(offset)})

	if err != nil {
		return resp.Err("Failed to get bots [row query] for generating RSS feed", err)
	}

	certBots := collector.CollectIDs(certBotIDs)

	for _, id := range certBots {
		err := state.SeoMapGenerator.AddToRss(d.Context, &fetchers.BotFetcher{}, &rssFeed, "Certified Bots", id)

		if err != nil {
			return resp.Err("Failed to add bot to RSS feed", err, zap.String("botId", id))
		}
	}

	// Get premium bots
	premiumBotIDs, err := q.GetPremiumBotIDs(d.Context, db.GetPremiumBotIDsParams{Limit: int32(limit), Offset: int32(offset)})

	if err != nil {
		return resp.Err("Failed to get bots [row query] for generating RSS feed", err)
	}

	premiumBots := collector.CollectIDs(premiumBotIDs)

	for _, id := range premiumBots {
		err := state.SeoMapGenerator.AddToRss(d.Context, &fetchers.BotFetcher{}, &rssFeed, "Premium Bots", id)

		if err != nil {
			return resp.Err("Failed to add bot to RSS feed", err, zap.String("botId", id))
		}
	}

	body, err := xml.Marshal(rssFeed)

	if err != nil {
		return resp.Err("Failed to marshal RSS feed", err)
	}

	_, err = state.Redis.Set(d.Context, "rssfeed-"+strconv.FormatUint(pageNum, 10), string(body), time.Minute*5).Result()

	if err != nil {
		// Log but dont error for this
		state.Logger.Error("Failed to set RSS feed cache", zap.Error(err))
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Bytes:  body,
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	}
}
