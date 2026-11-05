// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"

	"popplio/audit"
	"popplio/state"

	"go.uber.org/zap"
)

func AuditPrune(ctx context.Context) error {
	deleted, err := audit.Prune(ctx)

	if err != nil {
		return err
	}

	state.Logger.Info("audit_prune: done", zap.Int64("deleted", deleted))

	return nil
}
