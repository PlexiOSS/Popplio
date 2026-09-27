package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type verdict int

const (
	verdictApproved verdict = iota
	verdictDenied
	verdictUnverified
)

const verdictPushTTL = 7 * 24 * time.Hour

func entityName(ctx context.Context, targetType types.TargetType, targetID string) string {
	switch targetType {
	case types.TargetTypeServer:
		row, err := db.New(state.Pool).GetServerNameAndAvatar(ctx, targetID)
		if err == nil && strings.TrimSpace(row.Name) != "" {
			return row.Name
		}
	default:
		user, err := impls.GetPlatformUser(ctx, targetID)
		if err == nil && strings.TrimSpace(user.Username) != "" {
			return user.Username
		}
	}

	return ""
}

func verdictAlert(ctx context.Context, targetType types.TargetType, targetID string, v verdict, reason string) ptypes.Alert {
	label, path := "bot", "bots"
	if targetType == types.TargetTypeServer {
		label, path = "server", "servers"
	}

	subject := entityName(ctx, targetType, targetID)
	if subject == "" {
		subject = "Your " + label
	}

	reason = strings.TrimSpace(reason)

	alert := ptypes.Alert{
		URL:      pgtype.Text{String: fmt.Sprintf("%s/%s/%s", state.Config.Sites.Frontend, path, targetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
		Priority: ptypes.AlertPriorityMedium,
		PushTTL:  verdictPushTTL,
		AlertData: map[string]any{
			"target_type": string(targetType),
			"target_id":   targetID,
			"reason":      reason,
		},
	}

	switch v {
	case verdictApproved:
		alert.Type = ptypes.AlertTypeSuccess
		alert.Title = subject + " was approved"
		alert.Message = "It is now listed on Omniplex."
		if reason != "" {
			alert.Message += " Reviewer feedback: " + reason
		}
		alert.AlertData["verdict"] = "approved"
	case verdictDenied:
		alert.Type = ptypes.AlertTypeError
		alert.Title = subject + " was denied"
		alert.Message = "Reason: " + reason + " You can make changes and resubmit from your dashboard."
		alert.AlertData["verdict"] = "denied"
	case verdictUnverified:
		alert.Type = ptypes.AlertTypeWarning
		alert.Title = subject + " was sent back for review"
		alert.Message = "Reason: " + reason
		alert.AlertData["verdict"] = "unverified"
	}

	return alert
}

func notifyVerdict(ctx context.Context, owners []string, targetType types.TargetType, targetID string, v verdict, reason string) {
	if len(owners) == 0 {
		state.Logger.Warn("No owners to notify of review verdict", zap.String("targetID", targetID))
		return
	}

	impls.NotifyOwners(owners, verdictAlert(ctx, targetType, targetID, v, reason))
}
