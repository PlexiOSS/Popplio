package get_audit_events

import (
	"net/http"
	"strconv"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

func query(name, description string) docs.Parameter {
	return docs.Parameter{Name: name, Description: description, In: "query", Schema: docs.IdSchema}
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Get Activity Log",
		Description: "Lists activity log events, newest first. Needs `view_audit_logs`.",
		Params: []docs.Parameter{
			query("before", "Only events with a smaller ID, from next_before"),
			query("source", "api, panel, system, stripe or paypal"),
			query("actor_id", "Only events by this user, bot, server or team"),
			query("target_id", "Only events on this target"),
			query("action", "Only actions starting with this text, such as payment. or panel.UpdatePromoCodes"),
			{Name: "failed", Description: "true for only events with a status of 400 or more", In: "query", Schema: docs.BoolSchema},
			query("limit", "Events per page, 1 to 200. Default 50"),
		},
		Resp: types.AuditEventList{},
	}
}

func optText(r *http.Request, key string) pgtype.Text {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	return pgtype.Text{String: v, Valid: v != ""}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	staffPerms, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error while loading staff permissions", err)
	}

	if !staffPerms.Has(perms.StaffViewAuditLogs) {
		return resp.Forbidden("You do not have permission to view the activity log [view_audit_logs]")
	}

	params := db.ListAuditEventsParams{
		Source:   optText(r, "source"),
		ActorID:  optText(r, "actor_id"),
		TargetID: optText(r, "target_id"),
		Lim:      defaultLimit,
	}

	if action := optText(r, "action"); action.Valid {
		action.String = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(action.String)
		params.ActionPrefix = action
	}

	if r.URL.Query().Get("failed") == "true" {
		params.FailedOnly = pgtype.Bool{Bool: true, Valid: true}
	}

	if before := r.URL.Query().Get("before"); before != "" {
		id, err := strconv.ParseInt(before, 10, 64)

		if err != nil || id <= 0 {
			return resp.BadRequest("before must be an event ID")
		}

		params.Before = pgtype.Int8{Int64: id, Valid: true}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		n, err := strconv.Atoi(limit)

		if err != nil || n < 1 || n > maxLimit {
			return resp.BadRequest("limit must be between 1 and 200")
		}

		params.Lim = int32(n)
	}

	rows, err := db.New(state.Pool).ListAuditEvents(d.Context, params)

	if err != nil {
		return resp.Err("Failed to load the activity log", err)
	}

	showIP := staffPerms.Has(perms.StaffViewSensitive)
	out := types.AuditEventList{Events: make([]types.AuditEvent, 0, len(rows))}

	for _, row := range rows {
		event := types.AuditEvent{
			ID:        row.ID,
			CreatedAt: row.CreatedAt.Time,
			Source:    row.Source,
			ActorType: row.ActorType,
			Action:    row.Action,
			Data:      row.Data,
		}

		if row.ActorID.Valid {
			event.ActorID = &row.ActorID.String
		}

		if row.TargetType.Valid {
			event.TargetType = &row.TargetType.String
		}

		if row.TargetID.Valid {
			event.TargetID = &row.TargetID.String
		}

		if row.Status.Valid {
			status := int(row.Status.Int32)
			event.Status = &status
		}

		if showIP && row.Ip.Valid {
			event.IP = &row.Ip.String
		}

		if event.Data == nil {
			event.Data = map[string]any{}
		}

		out.Events = append(out.Events, event)
	}

	resolveNames(r, out.Events)

	if len(rows) == int(params.Lim) {
		next := strconv.FormatInt(rows[len(rows)-1].ID, 10)
		out.NextBefore = &next
	}

	return uapi.HttpResponse{Json: out}
}

func resolveNames(r *http.Request, events []types.AuditEvent) {
	ids := make([]string, 0, len(events)*2)
	seen := map[string]bool{}

	for _, e := range events {
		for _, id := range []*string{e.ActorID, e.TargetID} {
			if id != nil && *id != "" && !seen[*id] {
				seen[*id] = true
				ids = append(ids, *id)
			}
		}
	}

	if len(ids) == 0 {
		return
	}

	rows, err := db.New(state.Pool).ResolveAuditNames(r.Context(), ids)

	if err != nil {
		state.Logger.Warn("Failed to resolve activity log names", zap.Error(err))
		return
	}

	names := make(map[string]db.ResolveAuditNamesRow, len(rows))

	for _, row := range rows {
		if _, ok := names[row.ID]; !ok {
			names[row.ID] = row
		}
	}

	for i := range events {
		e := &events[i]

		if e.ActorID != nil {
			if n, ok := names[*e.ActorID]; ok {
				e.ActorName = &n.Name

				if n.Avatar != "" {
					e.ActorAvatar = &n.Avatar
				}
			}
		}

		if e.TargetID != nil {
			if n, ok := names[*e.TargetID]; ok {
				e.TargetName = &n.Name
			}
		}
	}
}
