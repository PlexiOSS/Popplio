package audit

import (
	"context"
	"encoding/json"
	"time"

	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	SourceAPI    = "api"
	SourcePanel  = "panel"
	SourceSystem = "system"
	SourceStripe = "stripe"
	SourcePayPal = "paypal"

	ActorUser      = "user"
	ActorStaff     = "staff"
	ActorSystem    = "system"
	ActorAnonymous = "anonymous"

	MaxDataBytes = 16 << 10
	Retention    = 180 * 24 * time.Hour
)

type Event struct {
	Source     string
	ActorType  string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Status     int
	IP         string
	Data       map[string]any
}

func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func Write(ctx context.Context, e Event) error {
	if state.Pool == nil {
		return nil
	}

	data := e.Data

	if data == nil {
		data = map[string]any{}
	}

	if raw, err := json.Marshal(data); err != nil || len(raw) > MaxDataBytes {
		data = map[string]any{"truncated": true, "bytes": len(raw)}
	}

	if e.ActorType == "" {
		e.ActorType = ActorAnonymous
	}

	return db.New(state.Pool).InsertAuditEvent(ctx, db.InsertAuditEventParams{
		Source:     e.Source,
		ActorType:  e.ActorType,
		ActorID:    text(e.ActorID),
		Action:     e.Action,
		TargetType: text(e.TargetType),
		TargetID:   text(e.TargetID),
		Status:     pgtype.Int4{Int32: int32(e.Status), Valid: e.Status != 0},
		Ip:         text(e.IP),
		Data:       data,
	})
}

func Record(e Event) {
	if state.Pool == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(state.Context, 10*time.Second)
		defer cancel()

		if err := Write(ctx, e); err != nil {
			state.Logger.Warn("Failed to record audit event", zap.Error(err), zap.String("action", e.Action))
		}
	}()
}

func System(action, targetType, targetID string, data map[string]any) {
	Record(Event{Source: SourceSystem, ActorType: ActorSystem, Action: action, TargetType: targetType, TargetID: targetID, Data: data})
}

func Prune(ctx context.Context) (int64, error) {
	return db.New(state.Pool).DeleteAuditEventsBefore(ctx, pgtype.Timestamptz{Time: time.Now().Add(-Retention), Valid: true})
}
