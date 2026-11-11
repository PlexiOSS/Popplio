// Copyright (C) 2026 NodeByte LTD

package views

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/listing"
	"popplio/state"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Event string

const (
	EventView   Event = "view"
	EventInvite Event = "invite"

	saltTTL = 48 * time.Hour
	seenTTL = 26 * time.Hour
)

var crawlers = regexp.MustCompile(`(?i)bot\b|bot/|crawl|spider|slurp|preview|headless|lighthouse|facebookexternalhit|embedly|python-|curl/|wget/|go-http-client|node-fetch|axios/|^$`)

func IsCrawler(userAgent string) bool {
	return crawlers.MatchString(strings.TrimSpace(userAgent))
}

func ViewerIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}

func SeenKey(salt, ip string, event Event, targetType, targetID string) string {
	sum := sha256.Sum256([]byte(salt + "|" + ip + "|" + string(event) + "|" + targetType + "|" + targetID))
	return "views:seen:" + hex.EncodeToString(sum[:])
}

func dailySalt(ctx context.Context, day string) (string, error) {
	key := "views:salt:" + day
	random := make([]byte, 32)

	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	if err := state.Redis.SetArgs(ctx, key, hex.EncodeToString(random), redis.SetArgs{Mode: "NX", TTL: saltTTL}).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return "", err
	}

	return state.Redis.Get(ctx, key).Result()
}

func firstToday(ctx context.Context, ip string, event Event, targetType, targetID string) (bool, error) {
	salt, err := dailySalt(ctx, time.Now().UTC().Format("2006-01-02"))

	if err != nil {
		return false, err
	}

	err = state.Redis.SetArgs(ctx, SeenKey(salt, ip, event, targetType, targetID), "1", redis.SetArgs{Mode: "NX", TTL: seenTTL}).Err()

	if errors.Is(err, redis.Nil) {
		return false, nil
	}

	return err == nil, err
}

func save(ctx context.Context, event Event, targetType, targetID string) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	switch {
	case event == EventView && targetType == "bot":
		err = q.UpdateBotClicks(ctx, targetID)
	case event == EventView:
		err = q.UpdateServerClicks(ctx, targetID)
	case targetType == "bot":
		err = q.UpdateBotInviteClicks(ctx, targetID)
	default:
		err = q.UpdateServerInviteClicks(ctx, targetID)
	}

	if err != nil {
		return err
	}

	if event == EventView {
		if err := q.RecordEntityView(ctx, db.RecordEntityViewParams{TargetType: targetType, TargetID: targetID}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func Docs(event Event, targetType string) func() *docs.Doc {
	summary := "Record " + targetType + " View"
	description := "Records that a visitor opened this " + targetType + "'s page. Each visitor counts once per " + targetType + " per day. Known crawlers, and people who manage the " + targetType + ", aren't counted. Signing in is optional. Always returns 204."

	if event == EventInvite {
		summary = "Record " + targetType + " Invite Click"
		description = "Records that a visitor opened this " + targetType + "'s invite. Each visitor counts once per " + targetType + " per day. Known crawlers, and people who manage the " + targetType + ", aren't counted. Signing in is optional. Always returns 204."
	}

	return func() *docs.Doc {
		return &docs.Doc{
			Summary:     summary,
			Description: description,
			Params: []docs.Parameter{
				{
					Name:        "id",
					Description: "The " + targetType + "'s ID",
					Required:    true,
					In:          "path",
					Schema:      docs.IdSchema,
				},
			},
		}
	}
}

func Route(event Event, targetType string) func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return func(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
		limit, err := ratelimit.Ratelimit{
			Expiry:      time.Minute,
			MaxRequests: 60,
			Bucket:      "listing_views",
		}.Limit(d.Context, r)

		if err != nil {
			return resp.Err("Error while ratelimiting", err)
		}

		if limit.Exceeded {
			return resp.RateLimited(limit)
		}

		done := uapi.DefaultResponse(http.StatusNoContent)

		if IsCrawler(r.UserAgent()) {
			return done
		}

		id := chi.URLParam(r, "id")

		public, err := listing.EntityIsPublic(d.Context, targetType, id)

		if err != nil || !public {
			return done
		}

		if d.Auth.Authorized && listing.CanViewUnlisted(d.Context, d.Auth, targetType, id) {
			return done
		}

		first, err := firstToday(d.Context, ViewerIP(r), event, targetType, id)

		if err != nil {
			state.Logger.Warn("views: failed to check for a repeat visitor", zap.Error(err))
			return done
		}

		if !first {
			return done
		}

		if err := save(d.Context, event, targetType, id); err != nil {
			state.Logger.Error("views: failed to record", zap.Error(err), zap.String("target_id", id), zap.String("event", string(event)))
		}

		return done
	}
}
