// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"

	"popplio/analytics"
)

func EntityStatsSnapshot(ctx context.Context) error {
	if _, err := analytics.Snapshot(ctx); err != nil {
		return fmt.Errorf("snapshotting listing stats: %w", err)
	}

	return nil
}
