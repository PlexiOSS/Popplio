// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"time"

	"popplio/db"
	"popplio/routes/servertemplates/assets"
	"popplio/state"

	"go.uber.org/zap"
)

const templateSyncBatch = 25

func TemplateSync(ctx context.Context) error {
	rows, err := db.New(state.Pool).ListServerTemplatesToSync(ctx, templateSyncBatch)

	if err != nil {
		return err
	}

	var synced, gone, failed int

	for i, row := range rows {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}

		_, err := assets.Sync(ctx, row.ID, row.Code)

		switch {
		case errors.Is(err, assets.ErrTemplateGone):
			gone++
		case err != nil:
			failed++
			state.Logger.Warn("template_sync: failed", zap.String("id", row.ID), zap.Error(err))
		default:
			synced++
		}
	}

	state.Logger.Info("template_sync: done", zap.Int("synced", synced), zap.Int("gone", gone), zap.Int("failed", failed))

	return nil
}
