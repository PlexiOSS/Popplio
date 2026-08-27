package panel

import (
	"context"
	"errors"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/notifications"
	"popplio/perms"
	"popplio/reports"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type reportRow struct {
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

func (r reportRow) toPanelReport(ctx context.Context) types.Report {
	name, url := r.TargetID, ""

	// A deleted target shouldn't break the report list — fall back to
	// showing the raw ID with no link.
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
		CreatedAt:   types.NewTimestamp(r.CreatedAt),
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

	// Every report operation, including listing, requires review_reports —
	// unlike Partners' public List, report contents (including reporter
	// identity) are staff-only by design.
	if !userPerms.Has(perms.StaffReviewReports) {
		return writeText(http.StatusForbidden, "You do not have permission to review reports [review_reports]"), nil
	}

	switch {
	case q.Action.List != nil:
		var rows pgx.Rows
		var err error

		if q.Action.List.Status != nil {
			rows, err = state.Pool.Query(ctx, "SELECT id, target_type, target_id, reporter_id, reason, description, status, resolved_by, resolution_note, created_at, resolved_at FROM reports WHERE status = $1 ORDER BY created_at DESC", *q.Action.List.Status)
		} else {
			rows, err = state.Pool.Query(ctx, "SELECT id, target_type, target_id, reporter_id, reason, description, status, resolved_by, resolution_note, created_at, resolved_at FROM reports ORDER BY created_at DESC")
		}

		if err != nil {
			return response{}, newError(err)
		}

		reportRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[reportRow])

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.Report, 0, len(reportRows))

		for _, row := range reportRows {
			out = append(out, row.toPanelReport(ctx))
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

	var reporterID string

	err := state.Pool.QueryRow(
		ctx,
		"UPDATE reports SET status = $1, resolved_by = $2, resolution_note = $3, resolved_at = NOW() WHERE id = $4 AND status IN ('open', 'under_review') RETURNING reporter_id",
		status,
		staffID,
		action.Note,
		action.ID,
	).Scan(&reporterID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return writeText(http.StatusNotFound, "Report not found, or has already been resolved/dismissed"), nil
		}

		return response{}, newError(err)
	}

	// Best-effort: the report is already resolved/dismissed at this point,
	// so a failure to notify the reporter shouldn't read back as the
	// resolution itself having failed.
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
