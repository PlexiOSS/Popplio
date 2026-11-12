// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"popplio/audit"
	"popplio/botroles"
	"popplio/state"

	"go.uber.org/zap"
)

type Task struct {
	Name        string
	Description string
	Enabled     bool
	Interval    time.Duration
	Run         func(ctx context.Context) error
}

func All() []Task {
	return []Task{
		{
			Name:        "bot_uptime_check",
			Description: "Recording uptime for every listed bot with a known presence (self-reported, main server, japi.rest or recent stats posts)",
			Enabled:     true,
			Interval:    5 * time.Minute,
			Run:         BotUptimeCheck,
		},
		{
			Name:        "moderation_scan",
			Description: "Re-running OpenAI moderation against listed bots/servers on a rolling basis and auto-filing a report on anything newly flagged",
			Enabled:     true,
			Interval:    30 * time.Minute,
			Run:         ModerationScan,
		},
		{
			Name:        "changelog_drafts",
			Description: "Creating unpublished public changelog drafts for new CHANGELOG.md releases",
			Enabled:     true,
			Interval:    5 * time.Minute,
			Run:         ChangelogDrafts,
		},
		{
			Name:        "stripe_reconcile",
			Description: "Fulfilling paid Stripe checkout sessions that the webhook missed",
			Enabled:     true,
			Interval:    time.Hour,
			Run:         StripeReconcile,
		},
		{
			Name:        "bot_presence_sync",
			Description: "Recording listed bots' Discord presence from Arcadia's servers, and from japi.rest for bots neither Omi nor Arcadia can see",
			Enabled:     true,
			Interval:    5 * time.Minute,
			Run:         BotPresenceSync,
		},
		{
			Name:        "user_cache_refresh",
			Description: "Refreshing cached Discord users (bots first, then listing owners/team members) before their cache entries expire",
			Enabled:     true,
			Interval:    5 * time.Minute,
			Run:         UserCacheRefresh,
		},
		{
			Name:        "auto_vote_watch",
			Description: "Reporting listings whose share of automatic votes jumped in the last day, for staff to check for Auto-vote abuse",
			Enabled:     true,
			Interval:    time.Hour,
			Run:         AutoVoteWatch,
		},
		{
			Name:        "template_sync",
			Description: "Refreshing server templates from Discord (usage count, channels, roles) and hiding ones that were deleted",
			Enabled:     true,
			Interval:    time.Hour,
			Run:         TemplateSync,
		},
		{
			Name:        "badge_sync",
			Description: "Awarding and revoking badges that have an automatic rule, then pushing changed linked roles to Discord",
			Enabled:     true,
			Interval:    time.Hour,
			Run:         BadgeSync,
		},
		{
			Name:        "credit_hold_sweep",
			Description: "Returning vote credits applied to premium checkouts that were never paid",
			Enabled:     true,
			Interval:    30 * time.Minute,
			Run:         CreditHoldSweep,
		},
		{
			Name:        "entity_stats_snapshot",
			Description: "Recording each listed bot and server's views, invite clicks and counts for the day so owners can see them over time",
			Enabled:     true,
			Interval:    time.Hour,
			Run:         EntityStatsSnapshot,
		},
		{
			Name:        "auto_vote",
			Description: "Casting automatic votes for auto-vote subscribers whose cooldown and schedule allow it",
			Enabled:     true,
			Interval:    5 * time.Minute,
			Run:         AutoVote,
		},
		{
			Name:        "auto_vote_summary",
			Description: "Sending the daily summary to Auto-vote subscribers who asked for one",
			Enabled:     true,
			Interval:    time.Hour,
			Run:         AutoVoteSummary,
		},
		{
			Name:        "credit_reminders",
			Description: "Telling owners, once before each monthly vote reset, about votes they can turn into credits and credits they haven't spent",
			Enabled:     true,
			Interval:    6 * time.Hour,
			Run:         CreditReminders,
		},
		{
			Name:        "bot_role_sync",
			Description: "Giving bots in the main server the roles they're missing: Bots, Certified Bots for certified listings and Partner Bots for partners",
			Enabled:     true,
			Interval:    15 * time.Minute,
			Run:         botroles.Sync,
		},
		{
			Name:        "booster_sync",
			Description: "Recording who boosts the main Discord server and applying their monthly booster premium",
			Enabled:     true,
			Interval:    10 * time.Minute,
			Run:         BoosterSync,
		},
		{
			Name:        "audit_prune",
			Description: "Deleting activity log events older than the retention period",
			Enabled:     true,
			Interval:    24 * time.Hour,
			Run:         AuditPrune,
		},
	}
}

func Start(ctx context.Context) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, task := range All() {
		if !task.Enabled {
			continue
		}

		wg.Add(1)

		go func(task Task) {
			defer wg.Done()
			runLoop(ctx, task)
		}(task)
	}

	return &wg
}

func runLoop(ctx context.Context, task Task) {
	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			state.Logger.Info("Task stopping", zap.String("task", task.Name))
			return
		case <-ticker.C:
			runOnce(ctx, task)
		}
	}
}

func runOnce(ctx context.Context, task Task) {
	defer func() {
		if rec := recover(); rec != nil {
			audit.System("task.panicked", "task", task.Name, map[string]any{"panic": fmt.Sprint(rec)})

			state.Logger.Error("Task panicked",
				zap.String("task", task.Name),
				zap.Any("panic", rec),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}()

	state.Logger.Info("Running task", zap.String("task", task.Name), zap.String("description", task.Description))

	start := time.Now()

	if err := task.Run(ctx); err != nil {
		audit.System("task.failed", "task", task.Name, map[string]any{"error": err.Error(), "took_ms": time.Since(start).Milliseconds()})
		state.Logger.Error("Task failed", zap.String("task", task.Name), zap.Error(err), zap.Duration("took", time.Since(start)))
		return
	}

	state.Logger.Info("Task finished", zap.String("task", task.Name), zap.Duration("took", time.Since(start)))
}
