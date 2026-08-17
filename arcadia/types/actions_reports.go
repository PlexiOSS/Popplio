package types

import "fmt"

// ReportAction is the union of report-review operations. Reports
// themselves are only ever created through Popplio's own public
// PUT /users/{uid}/{target_type}/{target_id}/reports — staff never create
// one, so unlike PartnerAction/BlogAction there is no Create variant here,
// only List/Resolve/Dismiss.
type ReportAction struct {
	List    *ReportListQuery
	Resolve *ReportResolve
	Dismiss *ReportResolve
}

// ReportListQuery filters the review queue. A nil Status returns every
// report regardless of status.
type ReportListQuery struct {
	Status *string `json:"status"`
}

// ReportResolve is shared by Resolve and Dismiss — both just move a report
// out of the open/under_review states with an optional staff note.
type ReportResolve struct {
	ID   string `json:"id"`
	Note string `json:"note"`
}

func (a *ReportAction) UnmarshalJSON(data []byte) error {
	*a = ReportAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("ReportAction: %w", err)
	}

	switch name {
	case "List":
		a.List = &ReportListQuery{}
		return decodeVariant("ReportAction", name, payload, a.List)
	case "Resolve":
		a.Resolve = &ReportResolve{}
		return decodeVariant("ReportAction", name, payload, a.Resolve)
	case "Dismiss":
		a.Dismiss = &ReportResolve{}
		return decodeVariant("ReportAction", name, payload, a.Dismiss)
	default:
		return errUnknownVariant("ReportAction", name)
	}
}

func (a ReportAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeVariant("List", a.List)
	case a.Resolve != nil:
		return encodeVariant("Resolve", a.Resolve)
	case a.Dismiss != nil:
		return encodeVariant("Dismiss", a.Dismiss)
	default:
		return nil, fmt.Errorf("ReportAction: no variant set")
	}
}

// Report is a single content report as shown in the staff panel. Unlike
// popplio/types.Report (the DB-facing shape, which never exposes
// ReporterID over JSON), this panel-facing shape deliberately does include
// the reporter — staff review is the one place identity is meant to be
// visible.
type Report struct {
	ID             string     `json:"id"`
	TargetType     string     `json:"target_type"`
	TargetID       string     `json:"target_id"`
	TargetName     string     `json:"target_name"`
	TargetURL      string     `json:"target_url"`
	ReporterID     string     `json:"reporter_id"`
	Reason         string     `json:"reason"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	ResolvedBy     *string    `json:"resolved_by"`
	ResolutionNote *string    `json:"resolution_note"`
	CreatedAt      Timestamp  `json:"created_at"`
	ResolvedAt     *Timestamp `json:"resolved_at"`
}
