// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"errors"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/notifications"
	"popplio/perms"
	"popplio/reports"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func toPanelReport(ctx context.Context, r db.Report) types.Report {
	name, url := r.TargetID, ""

	if info, err := reports.GetTargetInfo(ctx, state.Pool, r.TargetType, r.TargetID); err == nil {
		name, url = info.Name, info.URL
	}

	out := types.Report{
		ID:          impls.UUIDString(r.ID),
		TargetType:  r.TargetType,
		TargetID:    r.TargetID,
		TargetName:  name,
		TargetURL:   url,
		ReporterID:  r.ReporterID,
		Reason:      r.Reason,
		Description: r.Description,
		Status:      r.Status,
		CreatedAt:   types.NewTimestamp(r.CreatedAt.Time),
	}

	if r.ResolvedBy.Valid {
		out.ResolvedBy = &r.ResolvedBy.String
	}

	if r.ResolutionNote.Valid {
		out.ResolutionNote = &r.ResolutionNote.String
	}

	if r.ResolvedAt.Valid {
		ts := types.NewTimestamp(r.ResolvedAt.Time)
		out.ResolvedAt = &ts
	}

	return out
}

func (s *Server) updateReports(ctx context.Context, q *types.QUpdateReports) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !userPerms.Has(perms.StaffReviewReports) {
		return writeText(http.StatusForbidden, "You do not have permission to review reports [review_reports]"), nil
	}

	switch {
	case q.Action.List != nil:
		var status pgtype.Text

		if q.Action.List.Status != nil {
			status = pgtype.Text{String: *q.Action.List.Status, Valid: true}
		}

		reportRows, err := db.New(state.Pool).ListReports(ctx, status)

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.Report, 0, len(reportRows))

		for _, row := range reportRows {
			out = append(out, toPanelReport(ctx, row))
		}

		return writeJSON(http.StatusOK, out), nil
	case q.Action.Resolve != nil:
		return s.resolveReport(ctx, authData.UserID, q.Action.Resolve, "resolved")
	case q.Action.Dismiss != nil:
		return s.resolveReport(ctx, authData.UserID, q.Action.Dismiss, "dismissed")
	default:
		return writeText(http.StatusBadRequest, "No action specified"), nil
	}
}

func (s *Server) resolveReport(ctx context.Context, staffID string, action *types.ReportResolve, status string) (response, error) {
	if action.ID == "" {
		return writeText(http.StatusBadRequest, "id is required"), nil
	}

	var reportUUID pgtype.UUID

	if err := reportUUID.Scan(action.ID); err != nil {
		return writeText(http.StatusNotFound, "Report not found, or has already been resolved/dismissed"), nil
	}

	reporterID, err := db.New(state.Pool).ResolveReportReturningReporter(ctx, db.ResolveReportReturningReporterParams{
		Status:         status,
		ResolvedBy:     pgtype.Text{String: staffID, Valid: true},
		ResolutionNote: pgtype.Text{String: action.Note, Valid: true},
		ID:             reportUUID,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return writeText(http.StatusNotFound, "Report not found, or has already been resolved/dismissed"), nil
		}

		return response{}, newError(err)
	}

	alertTitle := "Report Dismissed"
	alertMessage := "Your report has been reviewed and dismissed."

	if status == "resolved" {
		alertTitle = "Report Resolved"
		alertMessage = "Your report has been reviewed and resolved."
	}

	if action.Note != "" {
		alertMessage += " Staff note: " + action.Note
	}

	if err := notifications.PushNotification(reporterID, ptypes.Alert{
		Type:     ptypes.AlertTypeInfo,
		Title:    alertTitle,
		Message:  alertMessage,
		Category: ptypes.AlertCategoryReports,
	}); err != nil {
		state.Logger.Warn("Failed to notify reporter of report resolution", zap.Error(err), zap.String("reportId", action.ID), zap.String("reporterId", reporterID))
	}

	return writeText(http.StatusOK, "OK"), nil
}
