package bgtasks

import (
	"context"
	"fmt"

	"popplio/moderation"
	"popplio/state"

	"go.uber.org/zap"
)

const moderationScanBatchSize = 25

type moderationScanRow struct {
	ID    string
	Short string
	Long  string
}

func ModerationScan(ctx context.Context) error {
	if state.Config.Meta.OpenAIAPIKey == "" {
		return nil
	}

	if err := scanTable(ctx, "bots", "bot_id", "bot"); err != nil {
		return fmt.Errorf("scanning bots: %w", err)
	}

	if err := scanTable(ctx, "servers", "server_id", "server"); err != nil {
		return fmt.Errorf("scanning servers: %w", err)
	}

	return nil
}

func scanTable(ctx context.Context, table, idColumn, targetType string) error {
	rows, err := state.Pool.Query(ctx,
		"SELECT "+idColumn+", short, long FROM "+table+" "+
			"WHERE type IN ('approved', 'certified', 'pending') "+
			"AND (moderation_checked_at IS NULL OR moderation_checked_at < NOW() - INTERVAL '7 days') "+
			"ORDER BY moderation_checked_at ASC NULLS FIRST "+
			"LIMIT $1",
		moderationScanBatchSize,
	)

	if err != nil {
		return fmt.Errorf("querying %s due for a moderation check: %w", table, err)
	}

	var due []moderationScanRow

	for rows.Next() {
		var row moderationScanRow

		if err := rows.Scan(&row.ID, &row.Short, &row.Long); err != nil {
			rows.Close()
			return fmt.Errorf("scanning %s row: %w", table, err)
		}

		due = append(due, row)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating %s due for a moderation check: %w", table, err)
	}

	for _, row := range due {
		result, err := moderation.CheckText(ctx, row.Short, row.Long)

		if err != nil {
			state.Logger.Error("moderation_scan: check failed, leaving unchecked for next run",
				zap.String("table", table), zap.String("id", row.ID), zap.Error(err))
			continue
		}

		if _, err := state.Pool.Exec(ctx,
			"UPDATE "+table+" SET moderation_flagged = $2, moderation_categories = $3, moderation_checked_at = NOW() WHERE "+idColumn+" = $1",
			row.ID, result.Flagged, result.Categories,
		); err != nil {
			return fmt.Errorf("%s=%s: storing moderation result: %w", idColumn, row.ID, err)
		}

		if result.Flagged {
			if err := moderation.FileAutoReport(ctx, targetType, row.ID, result.Categories); err != nil {
				state.Logger.Error("moderation_scan: failed to auto-file report",
					zap.String("table", table), zap.String("id", row.ID), zap.Error(err))
			}
		}
	}

	return nil
}
