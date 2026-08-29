// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type BotWhitelistRow struct {
	BotID     string    `db:"bot_id"`
	UserID    string    `db:"user_id"`
	Reason    string    `db:"reason"`
	CreatedAt time.Time `db:"created_at"`
}

type ReportRow struct {
	ID             pgtype.UUID      `db:"id"`
	TargetType     string           `db:"target_type"`
	TargetID       string           `db:"target_id"`
	ReporterID     string           `db:"reporter_id"`
	Reason         string           `db:"reason"`
	Description    string           `db:"description"`
	Status         string           `db:"status"`
	ResolvedBy     pgtype.Text      `db:"resolved_by"`
	ResolutionNote pgtype.Text      `db:"resolution_note"`
	CreatedAt      time.Time        `db:"created_at"`
	ResolvedAt     pgtype.Timestamp `db:"resolved_at"`
}

type RPCLogRow struct {
	ID        pgtype.UUID `db:"id"`
	UserID    string      `db:"user_id"`
	Method    string      `db:"method"`
	Data      []byte      `db:"data"`
	State     string      `db:"state"`
	CreatedAt time.Time   `db:"created_at"`
}

type DisciplinaryTypeRow struct {
	ID             string    `db:"id"`
	Name           string    `db:"name"`
	Description    string    `db:"description"`
	SelfAssignable bool      `db:"self_assignable"`
	PermLimits     []string  `db:"perm_limits"`
	Additory       bool      `db:"additory"`
	NeedsApproval  bool      `db:"needs_approval"`
	MaxExpiry      *float64  `db:"max_expiry"`
	CreatedAt      time.Time `db:"created_at"`
}
