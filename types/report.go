package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Report reasons. Kept as plain string constants (validated via
// `oneof=...` on the request DTO) rather than a Go enum type, matching the
// style of other enum-ish string fields in this codebase (e.g. Bot.Type).
const (
	ReportReasonLicenseViolation = "license_violation"
	ReportReasonTOSViolation     = "tos_violation"
	ReportReasonSpam             = "spam"
	ReportReasonOther            = "other"
)

// Report statuses, as staff move a report through the review queue.
const (
	ReportStatusOpen        = "open"
	ReportStatusUnderReview = "under_review"
	ReportStatusResolved    = "resolved"
	ReportStatusDismissed   = "dismissed"
)

// @ci table=reports
//
// Report represents a user-filed report against any votable entity (bots,
// servers, packs, etc, keyed the same (target_type, target_id) way
// entity_votes already is). Reporter identity is intentionally never
// exposed outside the staff panel — see routes/reports and
// arcadia/panel/ops_reports.go.
type Report struct {
	ID              pgtype.UUID      `db:"id" json:"id" description:"The report's ID"`
	TargetType      string           `db:"target_type" json:"target_type" description:"The type of entity being reported"`
	TargetID        string           `db:"target_id" json:"target_id" description:"The ID of the entity being reported"`
	ReporterID      string           `db:"reporter_id" json:"-" description:"The user ID of the reporter, staff-only, never sent to non-staff clients"`
	Reason          string           `db:"reason" json:"reason" description:"Why the entity was reported"`
	Description     string           `db:"description" json:"description" description:"Free-text detail from the reporter"`
	Status          string           `db:"status" json:"status" description:"The report's current review status"`
	ResolvedBy      pgtype.Text      `db:"resolved_by" json:"resolved_by" description:"The staff member who resolved/dismissed the report, if any"`
	ResolutionNote  pgtype.Text      `db:"resolution_note" json:"resolution_note" description:"A staff note left when resolving/dismissing the report"`
	CreatedAt       time.Time        `db:"created_at" json:"created_at" description:"When the report was filed"`
	ResolvedAt      pgtype.Timestamp `db:"resolved_at" json:"resolved_at" description:"When the report was resolved/dismissed, if it has been"`
}

// CreateReport is the request body for PUT /{target_type}/{target_id}/reports.
type CreateReport struct {
	Reason      string `json:"reason" validate:"required,oneof=license_violation tos_violation spam other" msg:"Reason must be one of license_violation, tos_violation, spam, or other"`
	Description string `json:"description" validate:"required,min=10,max=500" msg:"Description must be between 10 and 500 characters"`
}

// ReportStatCount is one (reason, status) bucket and how many reports fall
// into it. Deliberately the only thing GET /reports/stats ever returns —
// no report IDs, no target identity, no reporter identity. Public and
// anonymized by construction, not by filtering after the fact.
type ReportStatCount struct {
	Reason string `db:"reason" json:"reason"`
	Status string `db:"status" json:"status"`
	Count  int64  `db:"count" json:"count"`
}
