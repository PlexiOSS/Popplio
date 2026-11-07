// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"
	"strings"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/notifications"
	"popplio/perms"
	"popplio/state"
	ptypes "popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func pageImageURL(targetType, targetID, version string) string {
	return strings.TrimRight(state.Config.Sites.Frontend, "/") + "/cdn/backgrounds/" + targetType + "s/" + targetID + "/" + version + ".webp"
}

var notifyPageImage = func(ctx context.Context, targetType, targetID string, alert ptypes.Alert) {
	managerType := types.TargetTypeBot

	if targetType == "server" {
		managerType = types.TargetTypeServer
	}

	managers, err := impls.GetEntityManagers(ctx, managerType, targetID)

	if err != nil {
		state.Logger.Warn("Failed to find owners for a page background decision", zap.Error(err), zap.String("target_id", targetID))
		return
	}

	for _, owner := range managers.All() {
		if err := notifications.PushNotification(owner, alert); err != nil {
			state.Logger.Warn("Failed to notify an owner about a page background decision", zap.Error(err), zap.String("owner", owner))
		}
	}
}

func (s *Server) updatePageStyles(ctx context.Context, q *types.QUpdatePageStyles) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !userPerms.Has(perms.StaffReviewEntities) {
		return writeText(http.StatusForbidden, "You do not have permission to review page backgrounds [review_entities]"), nil
	}

	queries := db.New(state.Pool)

	switch {
	case q.Action.ListPending != nil:
		rows, err := queries.ListPendingPageImages(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.PendingPageImage, 0, len(rows))

		for _, r := range rows {
			item := types.PendingPageImage{
				TargetType:    r.TargetType,
				TargetID:      r.TargetID,
				Name:          r.TargetID,
				Version:       r.Version,
				ImageURL:      pageImageURL(r.TargetType, r.TargetID, r.Version),
				Dim:           int(r.Dim),
				Blur:          int(r.Blur),
				ImagePosition: r.ImagePosition,
				PendingSince:  types.NewTimestamp(r.PendingSince.Time),
			}

			if info, err := votes.GetEntityInfo(ctx, state.Pool, r.TargetID, r.TargetType); err == nil {
				item.Name = info.Name
				item.Avatar = info.Avatar
			}

			if r.ImageVersion.Valid {
				replaces := pageImageURL(r.TargetType, r.TargetID, r.ImageVersion.String)
				item.ReplacesURL = &replaces
			}

			out = append(out, item)
		}

		return writeJSON(http.StatusOK, out), nil
	case q.Action.Approve != nil:
		d := q.Action.Approve

		updated, err := queries.ApprovePageImage(ctx, db.ApprovePageImageParams{
			StaffID:    pgtype.Text{String: authData.UserID, Valid: true},
			TargetType: d.TargetType,
			TargetID:   d.TargetID,
			Version:    pgtype.Text{String: d.Version, Valid: true},
		})

		if err != nil {
			return response{}, newError(err)
		}

		if updated == 0 {
			return writeText(http.StatusConflict, "That background was already reviewed or replaced"), nil
		}

		notifyPageImage(ctx, d.TargetType, d.TargetID, ptypes.Alert{
			Type:     ptypes.AlertTypeSuccess,
			Title:    "Page background approved",
			Message:  "Your new page background is live.",
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/" + d.TargetType + "s/" + d.TargetID, Valid: true},
			Category: ptypes.AlertCategoryBotServerReviews,
		})

		return writeJSON(http.StatusOK, map[string]bool{"ok": true}), nil
	case q.Action.Reject != nil:
		d := q.Action.Reject
		reason := strings.TrimSpace(d.Reason)

		if len(reason) < 5 || len(reason) > 300 {
			return writeText(http.StatusBadRequest, "Give a reason between 5 and 300 characters, so the owner knows what to change"), nil
		}

		updated, err := queries.RejectPageImage(ctx, db.RejectPageImageParams{
			Reason:     pgtype.Text{String: reason, Valid: true},
			StaffID:    pgtype.Text{String: authData.UserID, Valid: true},
			TargetType: d.TargetType,
			TargetID:   d.TargetID,
			Version:    pgtype.Text{String: d.Version, Valid: true},
		})

		if err != nil {
			return response{}, newError(err)
		}

		if updated == 0 {
			return writeText(http.StatusConflict, "That background was already reviewed or replaced"), nil
		}

		notifyPageImage(ctx, d.TargetType, d.TargetID, ptypes.Alert{
			Type:     ptypes.AlertTypeWarning,
			Title:    "Page background not approved",
			Message:  "Your new page background wasn't approved: " + reason,
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/dashboard/" + d.TargetType + "s/" + d.TargetID + "?section=style", Valid: true},
			Category: ptypes.AlertCategoryBotServerReviews,
		})

		return writeJSON(http.StatusOK, map[string]bool{"ok": true}), nil
	}

	return writeText(http.StatusBadRequest, "Unknown page style action"), nil
}
