// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"

	"popplio/db"
	"popplio/moderation"
	"popplio/state"

	"go.uber.org/zap"
)

const moderationScanBatchSize = 25

type moderationScanRow struct {
	ID    string
	Short string
	Long  string
	NSFW  bool
}

func ModerationScan(ctx context.Context) error {
	if state.Config.Meta.OpenAIAPIKey == "" {
		return nil
	}

	if err := scanBots(ctx); err != nil {
		return fmt.Errorf("scanning bots: %w", err)
	}

	if err := scanServers(ctx); err != nil {
		return fmt.Errorf("scanning servers: %w", err)
	}

	return nil
}

func scanBots(ctx context.Context) error {
	q := db.New(state.Pool)

	rows, err := q.GetBotsDueForModerationScan(ctx, moderationScanBatchSize)

	if err != nil {
		return fmt.Errorf("querying bots due for a moderation check: %w", err)
	}

	due := make([]moderationScanRow, len(rows))
	for i, row := range rows {
		due[i] = moderationScanRow{ID: row.BotID, Short: row.Short, Long: row.Long, NSFW: row.Nsfw}
	}

	for _, row := range due {
		result, err := moderation.CheckText(ctx, row.Short, row.Long)

		if err != nil {
			state.Logger.Error("moderation_scan: check failed, leaving unchecked for next run",
				zap.String("table", "bots"), zap.String("id", row.ID), zap.Error(err))
			continue
		}

		result = moderation.EffectiveResult(result, row.NSFW)

		if err := q.RecordBotModerationScan(ctx, db.RecordBotModerationScanParams{
			BotID:                row.ID,
			ModerationFlagged:    result.Flagged,
			ModerationCategories: result.Categories,
		}); err != nil {
			return fmt.Errorf("bot_id=%s: storing moderation result: %w", row.ID, err)
		}

		if result.Flagged {
			if err := moderation.FileAutoReport(ctx, "bot", row.ID, result.Categories); err != nil {
				state.Logger.Error("moderation_scan: failed to auto-file report",
					zap.String("table", "bots"), zap.String("id", row.ID), zap.Error(err))
			}
		}
	}

	return nil
}

func scanServers(ctx context.Context) error {
	q := db.New(state.Pool)

	rows, err := q.GetServersDueForModerationScan(ctx, moderationScanBatchSize)

	if err != nil {
		return fmt.Errorf("querying servers due for a moderation check: %w", err)
	}

	due := make([]moderationScanRow, len(rows))
	for i, row := range rows {
		due[i] = moderationScanRow{ID: row.ServerID, Short: row.Short, Long: row.Long, NSFW: row.Nsfw}
	}

	for _, row := range due {
		result, err := moderation.CheckText(ctx, row.Short, row.Long)

		if err != nil {
			state.Logger.Error("moderation_scan: check failed, leaving unchecked for next run",
				zap.String("table", "servers"), zap.String("id", row.ID), zap.Error(err))
			continue
		}

		result = moderation.EffectiveResult(result, row.NSFW)

		if err := q.RecordServerModerationScan(ctx, db.RecordServerModerationScanParams{
			ServerID:             row.ID,
			ModerationFlagged:    result.Flagged,
			ModerationCategories: result.Categories,
		}); err != nil {
			return fmt.Errorf("server_id=%s: storing moderation result: %w", row.ID, err)
		}

		if result.Flagged {
			if err := moderation.FileAutoReport(ctx, "server", row.ID, result.Categories); err != nil {
				state.Logger.Error("moderation_scan: failed to auto-file report",
					zap.String("table", "servers"), zap.String("id", row.ID), zap.Error(err))
			}
		}
	}

	return nil
}
