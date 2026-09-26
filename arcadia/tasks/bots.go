package tasks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/japi"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/disgoorg/disgo/discord"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/dovewing"
)

type unclaimNotification struct {
	BotID       string
	ClaimedBy   string
	LastClaimed time.Time
}

func AutoUnclaim(ctx context.Context) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	q := db.New(tx)

	bots, err := q.GetStaleClaimedBotsForUpdate(ctx)

	if err != nil {
		return fmt.Errorf("Error while checking for claimed bots: %s", err)
	}

	var notifications []unclaimNotification

	for _, bot := range bots {
		state.Logger.Info("Unclaiming bot", zap.String("botID", bot.BotID))

		if err := q.ResubmitBot(ctx, bot.BotID); err != nil {
			return fmt.Errorf("Error while unclaiming bot %s: %s", bot.BotID, err)
		}

		// Only bots with a known reviewer and claim time get announced.
		if !bot.ClaimedBy.Valid || !bot.LastClaimed.Valid {
			continue
		}

		notifications = append(notifications, unclaimNotification{
			BotID:       bot.BotID,
			ClaimedBy:   bot.ClaimedBy.String,
			LastClaimed: bot.LastClaimed.Time,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Error while committing transaction: %s", err)
	}

	for _, n := range notifications {
		err := impls.SendChannel(state.Config.Channels.TestingLounge, discord.MessageCreate{
			Content: fmt.Sprintf("<@%s>", n.ClaimedBy),
			Embeds: []discord.Embed{{
				Title: "Auto-Unclaimed Bot",
				Description: fmt.Sprintf(
					"Bot <@%s> was auto-unclaimed (was previously claimed by <@%s> due to it being claimed for over one hour without being approved or denied).\nThis bot was last claimed <t:%d:R>.",
					n.BotID, n.ClaimedBy, n.LastClaimed.Unix()),
				Color: impls.ColourRed,
			}},
		})

		if err != nil {
			return fmt.Errorf("Error while sending message in #lounge: %s", err)
		}

		owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, n.BotID)

		if err != nil {
			return err
		}

		err = impls.SendModLog(discord.MessageCreate{
			Content: owners.MentionUsers(),
			Embeds: []discord.Embed{{
				Title: "Bot Unclaimed!",
				Description: fmt.Sprintf(`
<@%s> has been unclaimed as it was not being actively reviewed.

Don't worry, this is normal, could just be our staff looking more into your bots functionality!

For more information, you can contact the current reviewer <@%s>

*This bot was claimed <t:%d:R>. This is a automated message letting you know about whats going on...*
                            `, n.BotID, n.ClaimedBy, n.LastClaimed.Unix()),
				Footer: impls.Footer("This is completely normal, don't worry!"),
			}},
		})

		if err != nil {
			return fmt.Errorf("Error while sending message in #mod-logs: %s", err)
		}

		impls.NotifyOwners(owners.All(), ptypes.Alert{
			Type:     ptypes.AlertTypeInfo,
			Title:    "Bot Review Paused",
			Message:  "Your bot was claimed for review by staff but wasn't approved or denied within an hour, so it's been unclaimed and will be picked back up soon.",
			URL:      pgtype.Text{String: fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, n.BotID), Valid: true},
			Category: ptypes.AlertCategoryBotServerReviews,
		})
	}

	return nil
}

// DeletedBots removes bots whose Discord application no longer exists.
func DeletedBots(ctx context.Context) error {
	q := db.New(state.Pool)

	botIDs, err := q.GetAllBotIDs(ctx)

	if err != nil {
		return fmt.Errorf("Error while fetching all bots: %s", err)
	}

	for _, botID := range botIDs {
		username, err := q.GetCachedDiscordUsername(ctx, botID)

		if err != nil {
			state.Logger.Warn("Bot is not in internal_user_cache__discord, forcing indexing of bot", zap.String("botID", botID))

			// Ask Popplio to index the user, then skip this round.
			url := fmt.Sprintf("%s/platform/user/%s?platform=discord", state.Config.Sites.API, botID)

			resp, err := httpGet(ctx, url)

			if err != nil {
				state.Logger.Error("Failed to fetch bot from Popplio", zap.String("botID", botID), zap.Error(err))
				continue
			}

			resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				state.Logger.Error("Failed to fetch bot from Popplio", zap.String("botID", botID), zap.Int("status", resp.StatusCode))
			}

			continue
		}

		if !strings.HasPrefix(username, "Deleted User") && !strings.HasPrefix(username, "deleted_user") {
			continue
		}

		state.Logger.Info("Bot is potentially deleted, checking with Discord API", zap.String("botID", botID))

		url := fmt.Sprintf("%s/api/v10/applications/%s/rpc", state.Config.Meta.PopplioProxy, botID)

		resp, err := httpGet(ctx, url)

		if err != nil {
			return fmt.Errorf("Error while fetching RPC endpoint for bot %s: %s", botID, err)
		}

		status := resp.StatusCode
		resp.Body.Close()

		if status >= 200 && status <= 299 {
			// Bot still exists.
			continue
		}

		state.Logger.Info("Bot is deleted from Discord, removing from database", zap.String("botID", botID))

		owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, botID)

		if err != nil {
			return err
		}

		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return fmt.Errorf("Error creating transaction: %s", err)
		}

		if err := db.New(tx).DeleteBotByID(ctx, botID); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("Error while deleting bot %s from database: %s", botID, err)
		}

		err = impls.SendModLog(discord.MessageCreate{
			Content: owners.MentionUsers(),
			Embeds: []discord.Embed{{
				Title:       "Bot Deleted From Discord!",
				URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, botID),
				Description: fmt.Sprintf("`%s` has been deleted from Discord, and so will be removed from list!", botID),
				Fields: []discord.EmbedField{
					{Name: "Bot", Value: botID, Inline: impls.InlineTrue()},
				},
				Footer: impls.Footer("If this is a mistake, please contact support!"),
				Color:  impls.ColourGreen,
			}},
		})

		if err != nil {
			tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		impls.NotifyOwners(owners.All(), ptypes.Alert{
			Type:     ptypes.AlertTypeWarning,
			Title:    "Bot Removed From Listing",
			Message:  "Your bot `" + botID + "` was deleted from Discord, so it's been removed from the listing. If this is a mistake, please contact support.",
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/support", Valid: true},
			Category: ptypes.AlertCategoryBotServerReviews,
		})
	}

	return nil
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

// PremiumRemove strips premium from bots that lost approval or whose
// subscription expired.
func PremiumRemove(ctx context.Context) error {
	q := db.New(state.Pool)

	botRows, err := q.GetExpiredPremiumBots(ctx)

	if err != nil {
		return fmt.Errorf("Error while checking for expired premium bots: %s", err)
	}

	for _, bot := range botRows {
		user, err := impls.GetPlatformUser(ctx, bot.BotID)

		if err != nil {
			return err
		}

		state.Logger.Info("Removing premium from bot", zap.String("botID", bot.BotID))

		if err := q.RemoveBotPremium(ctx, bot.BotID); err != nil {
			return fmt.Errorf("Error while removing premium from bot %s: %s", bot.BotID, err)
		}

		owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, bot.BotID)

		if err != nil {
			return err
		}

		var msg string

		if bot.Type != "approved" && bot.Type != "certified" {
			msg = fmt.Sprintf(
				"<@%s> (%s) by %s has been removed from the premium list because it is not/no longer approved or certified.",
				bot.BotID, user.Username, owners.MentionUsers())
		} else {
			msg = fmt.Sprintf(
				"<@%s> (%s) by %s has been removed from the premium list as their subscription has expired.",
				bot.BotID, user.Username, owners.MentionUsers())
		}

		if err := impls.SendModLog(discord.MessageCreate{Content: msg}); err != nil {
			return err
		}

		expiredReason := "your subscription has expired"

		if bot.Type != "approved" && bot.Type != "certified" {
			expiredReason = "it is not/no longer approved or certified"
		}

		impls.NotifyOwners(owners.All(), ptypes.Alert{
			Type:     ptypes.AlertTypeWarning,
			Title:    "Premium Removed",
			Message:  "Premium has been removed from " + user.Username + " because " + expiredReason + ".",
			URL:      pgtype.Text{String: fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, bot.BotID), Valid: true},
			Category: ptypes.AlertCategoryPayments,
		})
	}

	return nil
}

var (
	japiReqsMade    atomic.Int64
	japiLastRefresh atomic.Int64
)

const japiHourlyBudget = 1800

func JapiUpdater(ctx context.Context) error {
	now := time.Now().Unix()

	if now-japiLastRefresh.Load() >= 3600 {
		japiReqsMade.Store(0)
		japiLastRefresh.Store(now)
	}

	q := db.New(state.Pool)

	botIDs, err := q.GetBotsDueForJapiUpdate(ctx)

	if err != nil {
		return err
	}

	for _, botID := range botIDs {
		if japiReqsMade.Add(1) > japiHourlyBudget {
			return errors.New("Internal error: JAPI rate limit hit")
		}

		app, err := japi.GetApplication(ctx, botID)

		if errors.Is(err, japi.ErrRateLimited) || errors.Is(err, japi.ErrUnavailable) {
			return err
		}

		if err != nil && !errors.Is(err, japi.ErrNotFound) {
			state.Logger.Error("Failed to fetch bot from JAPI", zap.String("botID", botID), zap.Error(err))
			continue
		}

		if app != nil && app.Bot != nil && app.Bot.ApproximateGuildCount != nil {
			err = q.UpdateBotJapiServers(ctx, db.UpdateBotJapiServersParams{
				Servers: *app.Bot.ApproximateGuildCount,
				BotID:   botID,
			})
		} else {
			err = q.TouchBotJapiUpdate(ctx, botID)
		}

		if err != nil {
			state.Logger.Error("Failed to save JAPI update for bot", zap.String("botID", botID), zap.Error(err))
			continue
		}

		if app != nil && app.Bot != nil {
			refreshBotProfileIfChanged(ctx, botID, app.Bot.Username, app.Bot.Avatar)
		}
	}

	return nil
}

var discordAvatarHashRe = regexp.MustCompile(`/avatars/[0-9]+/([A-Za-z0-9_]+)`)

func refreshBotProfileIfChanged(ctx context.Context, botID, username string, avatarHash *string) {
	cached, err := dovewing.GetUser(ctx, botID, state.DovewingPlatformDiscord)

	if err != nil {
		return
	}

	var cachedHash string
	if m := discordAvatarHashRe.FindStringSubmatch(cached.Avatar); m != nil {
		cachedHash = m[1]
	}

	var japiHash string
	if avatarHash != nil {
		japiHash = *avatarHash
	}

	if cached.Username == username && cachedHash == japiHash {
		return
	}

	state.Logger.Info("Bot profile changed upstream, refreshing cached user", zap.String("botID", botID))

	if _, err := dovewing.ClearUser(ctx, botID, state.DovewingPlatformDiscord, dovewing.ClearUserReq{}); err != nil {
		state.Logger.Warn("Failed to clear cached bot user", zap.String("botID", botID), zap.Error(err))
		return
	}

	if _, err := dovewing.GetUser(ctx, botID, state.DovewingPlatformDiscord); err != nil {
		state.Logger.Warn("Failed to re-resolve bot user after clearing cache", zap.String("botID", botID), zap.Error(err))
	}
}
