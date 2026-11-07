// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/notifications"
	"popplio/perms"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	maxDisciplinaryTitle       = 120
	maxDisciplinaryDescription = 2000
)

func (s *Server) updateDisciplinaries(ctx context.Context, q *types.QUpdateDisciplinaries) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	actorID := authData.UserID

	switch {
	case q.Action.List != nil:
		return s.listDisciplinaries(ctx, actorID, userPerms, q.Action.List)
	case q.Action.Issue != nil:
		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to issue disciplinaries [manage_disciplinaries]"), nil
		}

		a := q.Action.Issue

		if a.UserID == actorID {
			return writeText(http.StatusBadRequest, "You can't issue a disciplinary to yourself. Request a self-assignable one from your Account page instead."), nil
		}

		if strings.TrimSpace(a.Description) == "" {
			return writeText(http.StatusBadRequest, "Give a reason for the disciplinary"), nil
		}

		if resp, ok := s.requireOutranks(ctx, actorID, a.UserID, "issue a disciplinary to"); !ok {
			return resp, nil
		}

		return s.createDisciplinary(ctx, actorID, a.UserID, a.Type, a.Title, a.Description, a.Expiry, false)
	case q.Action.Request != nil:
		a := q.Action.Request
		return s.createDisciplinary(ctx, actorID, actorID, a.Type, a.Title, a.Description, a.Expiry, true)
	case q.Action.Approve != nil:
		return s.decideDisciplinary(ctx, actorID, userPerms, q.Action.Approve, true)
	case q.Action.Deny != nil:
		return s.decideDisciplinary(ctx, actorID, userPerms, q.Action.Deny, false)
	case q.Action.Revoke != nil:
		return s.revokeDisciplinary(ctx, actorID, userPerms, q.Action.Revoke)
	default:
		return writeText(http.StatusBadRequest, "No action specified"), nil
	}
}

func (s *Server) listDisciplinaries(ctx context.Context, actorID string, userPerms perms.Set, a *types.DisciplinaryList) (response, error) {
	queries := db.New(state.Pool)

	if a.UserID != nil && *a.UserID == actorID {
		out, err := impls.GetStaffDisciplinaries(ctx, actorID, false)

		if err != nil {
			return response{}, newError(err)
		}

		return writeJSON(http.StatusOK, out), nil
	}

	if !userPerms.Has(perms.StaffManageDisciplinaries) {
		return writeText(http.StatusForbidden, "You do not have permission to view other staff members' disciplinaries [manage_disciplinaries]"), nil
	}

	if a.UserID != nil {
		out, err := impls.GetStaffDisciplinaries(ctx, *a.UserID, false)

		if err != nil {
			return response{}, newError(err)
		}

		return writeJSON(http.StatusOK, out), nil
	}

	var stateFilter pgtype.Text

	if a.State != nil && *a.State != "" {
		stateFilter = pgtype.Text{String: *a.State, Valid: true}
	}

	rows, err := queries.ListAllStaffDisciplinaries(ctx, stateFilter)

	if err != nil {
		return response{}, newError(err)
	}

	out := make([]types.StaffDisciplinary, 0, len(rows))

	for _, row := range rows {
		d, err := impls.DisciplinaryFromRow(db.GetStaffDisciplinariesRow(row))

		if err != nil {
			continue
		}

		out = append(out, d)
	}

	return writeJSON(http.StatusOK, out), nil
}

func (s *Server) requireOutranks(ctx context.Context, actorID, targetID, verb string) (response, bool) {
	actor, err := perms.LoadStaff(ctx, actorID)

	if err != nil {
		return writeText(http.StatusInternalServerError, err.Error()), false
	}

	target, err := perms.LoadStaff(ctx, targetID)

	if errors.Is(err, pgx.ErrNoRows) {
		return writeText(http.StatusNotFound, "That user isn't a staff member"), false
	}

	if err != nil {
		return writeText(http.StatusInternalServerError, err.Error()), false
	}

	if target.BotAccount {
		return writeText(http.StatusForbidden, perms.ErrBotAccount.Error()), false
	}

	if actor.Rank() >= target.Rank() {
		return writeText(http.StatusForbidden, fmt.Sprintf("You need to rank above them to %s them", verb)), false
	}

	return response{}, true
}

func findDisciplinaryType(ctx context.Context, id string) (db.ListStaffDisciplinaryTypesRow, bool, error) {
	rows, err := db.New(state.Pool).ListStaffDisciplinaryTypes(ctx)

	if err != nil {
		return db.ListStaffDisciplinaryTypesRow{}, false, err
	}

	for _, row := range rows {
		if row.ID == id {
			return row, true, nil
		}
	}

	return db.ListStaffDisciplinaryTypesRow{}, false, nil
}

func validateExpiry(expiry *int64, maxExpiry *float64) (string, bool) {
	if expiry != nil && *expiry <= 0 {
		return "The length must be more than zero", false
	}

	if maxExpiry == nil {
		return "", true
	}

	if expiry == nil {
		return "This type needs a length", false
	}

	if float64(*expiry) > *maxExpiry {
		return fmt.Sprintf("This type can last at most %s", humanDuration(int64(*maxExpiry))), false
	}

	return "", true
}

func humanDuration(secs int64) string {
	d := time.Duration(secs) * time.Second

	switch {
	case d%(24*time.Hour) == 0:
		days := int64(d / (24 * time.Hour))
		return fmt.Sprintf("%d day%s", days, plural(days))
	case d%time.Hour == 0:
		hours := int64(d / time.Hour)
		return fmt.Sprintf("%d hour%s", hours, plural(hours))
	default:
		mins := int64(d / time.Minute)
		return fmt.Sprintf("%d minute%s", mins, plural(mins))
	}
}

func plural(n int64) string {
	if n == 1 {
		return ""
	}

	return "s"
}

func (s *Server) createDisciplinary(ctx context.Context, actorID, targetID, typeID, title, description string, expiry *int64, self bool) (response, error) {
	dtype, ok, err := findDisciplinaryType(ctx, typeID)

	if err != nil {
		return response{}, newError(err)
	}

	if !ok {
		return writeText(http.StatusNotFound, "That disciplinary type doesn't exist"), nil
	}

	if self && !dtype.SelfAssignable {
		return writeText(http.StatusForbidden, "You can't give yourself this type"), nil
	}

	if msg, ok := validateExpiry(expiry, numericToFloat64Ptr(dtype.MaxExpiry)); !ok {
		return writeText(http.StatusBadRequest, msg), nil
	}

	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	if title == "" {
		title = dtype.Name
	}

	if len(title) > maxDisciplinaryTitle || len(description) > maxDisciplinaryDescription {
		return writeText(http.StatusBadRequest, fmt.Sprintf("The title can be at most %d characters and the reason at most %d", maxDisciplinaryTitle, maxDisciplinaryDescription)), nil
	}

	queries := db.New(state.Pool)

	if self {
		existing, err := impls.GetStaffDisciplinaries(ctx, targetID, false)

		if err != nil {
			return response{}, newError(err)
		}

		for _, d := range existing {
			if d.Type.ID == typeID && (d.State == "pending" || d.State == "active") {
				return writeText(http.StatusConflict, "You already have one of these pending or active"), nil
			}
		}
	}

	stateValue := "active"
	startsAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	if dtype.NeedsApproval {
		stateValue = "pending"
		startsAt = pgtype.Timestamptz{}
	}

	var expirySecs pgtype.Int8

	if expiry != nil {
		expirySecs = pgtype.Int8{Int64: *expiry, Valid: true}
	}

	id, err := queries.InsertStaffDisciplinary(ctx, db.InsertStaffDisciplinaryParams{
		UserID:      targetID,
		Title:       title,
		Description: description,
		Type:        typeID,
		State:       stateValue,
		ExpirySecs:  expirySecs,
		IssuedBy:    pgtype.Text{String: actorID, Valid: true},
		StartsAt:    startsAt,
	})

	if err != nil {
		return response{}, newError(err)
	}

	d, err := loadDisciplinary(ctx, id)

	if err != nil {
		return response{}, newError(err)
	}

	if !self {
		message := fmt.Sprintf("You've been given a %s: %s", dtype.Name, title)

		if stateValue == "pending" {
			message = fmt.Sprintf("A %s has been issued to you and is waiting for approval: %s", dtype.Name, title)
		}

		notifyDisciplinary(targetID, "Staff disciplinary", message, ptypes.AlertTypeWarning)
	}

	return writeJSON(http.StatusOK, d), nil
}

func loadDisciplinary(ctx context.Context, id pgtype.UUID) (types.StaffDisciplinary, error) {
	row, err := db.New(state.Pool).GetStaffDisciplinaryByID(ctx, id)

	if err != nil {
		return types.StaffDisciplinary{}, err
	}

	return impls.DisciplinaryFromRow(db.GetStaffDisciplinariesRow(row))
}

func parseDisciplinaryID(id string) (pgtype.UUID, bool) {
	var u pgtype.UUID

	if err := u.Scan(id); err != nil {
		return u, false
	}

	return u, true
}

func (s *Server) decideDisciplinary(ctx context.Context, actorID string, userPerms perms.Set, a *types.DisciplinaryDecision, approve bool) (response, error) {
	if !userPerms.Has(perms.StaffManageDisciplinaries) {
		return writeText(http.StatusForbidden, "You do not have permission to approve or deny disciplinaries [manage_disciplinaries]"), nil
	}

	id, ok := parseDisciplinaryID(a.ID)

	if !ok {
		return writeText(http.StatusNotFound, "Disciplinary not found"), nil
	}

	d, err := loadDisciplinary(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return writeText(http.StatusNotFound, "Disciplinary not found"), nil
	}

	if err != nil {
		return response{}, newError(err)
	}

	if d.State != "pending" {
		return writeText(http.StatusConflict, "This isn't waiting for approval"), nil
	}

	if d.UserID == actorID {
		return writeText(http.StatusForbidden, "You can't approve or deny your own"), nil
	}

	if d.IssuedBy != nil && *d.IssuedBy == actorID {
		return writeText(http.StatusForbidden, "Someone other than the person who issued it has to approve or deny it"), nil
	}

	if resp, ok := s.requireOutranks(ctx, actorID, d.UserID, "approve or deny disciplinaries for"); !ok {
		return resp, nil
	}

	newState := "denied"

	if approve {
		newState = "active"
	}

	var note pgtype.Text

	if n := strings.TrimSpace(a.Note); n != "" {
		note = pgtype.Text{String: n, Valid: true}
	}

	n, err := db.New(state.Pool).DecideStaffDisciplinary(ctx, db.DecideStaffDisciplinaryParams{
		State:        newState,
		DecidedBy:    pgtype.Text{String: actorID, Valid: true},
		DecisionNote: note,
		ID:           id,
	})

	if err != nil {
		return response{}, newError(err)
	}

	if n == 0 {
		return writeText(http.StatusConflict, "This isn't waiting for approval"), nil
	}

	if approve {
		notifyDisciplinary(d.UserID, "Staff disciplinary approved", fmt.Sprintf("Your %s was approved and is now in effect: %s", d.Type.Name, d.Title), ptypes.AlertTypeInfo)
	} else {
		notifyDisciplinary(d.UserID, "Staff disciplinary denied", fmt.Sprintf("Your %s was denied: %s", d.Type.Name, d.Title), ptypes.AlertTypeInfo)
	}

	updated, err := loadDisciplinary(ctx, id)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, updated), nil
}

func (s *Server) revokeDisciplinary(ctx context.Context, actorID string, userPerms perms.Set, a *types.DisciplinaryDecision) (response, error) {
	id, ok := parseDisciplinaryID(a.ID)

	if !ok {
		return writeText(http.StatusNotFound, "Disciplinary not found"), nil
	}

	d, err := loadDisciplinary(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return writeText(http.StatusNotFound, "Disciplinary not found"), nil
	}

	if err != nil {
		return response{}, newError(err)
	}

	if d.State != "pending" && d.State != "active" {
		return writeText(http.StatusConflict, "This has already ended"), nil
	}

	ownSelfAssigned := d.UserID == actorID && d.Type.SelfAssignable
	ownPendingIssue := d.State == "pending" && d.IssuedBy != nil && *d.IssuedBy == actorID

	if !ownSelfAssigned && !ownPendingIssue {
		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to lift disciplinaries [manage_disciplinaries]"), nil
		}

		if d.UserID == actorID {
			return writeText(http.StatusForbidden, "Someone who ranks above you has to lift this"), nil
		}

		if resp, ok := s.requireOutranks(ctx, actorID, d.UserID, "lift disciplinaries for"); !ok {
			return resp, nil
		}
	}

	var reason pgtype.Text

	if r := strings.TrimSpace(a.Note); r != "" {
		reason = pgtype.Text{String: r, Valid: true}
	}

	n, err := db.New(state.Pool).RevokeStaffDisciplinary(ctx, db.RevokeStaffDisciplinaryParams{
		RevokedBy:    pgtype.Text{String: actorID, Valid: true},
		RevokeReason: reason,
		ID:           id,
	})

	if err != nil {
		return response{}, newError(err)
	}

	if n == 0 {
		return writeText(http.StatusConflict, "This has already ended"), nil
	}

	if d.UserID != actorID {
		notifyDisciplinary(d.UserID, "Staff disciplinary lifted", fmt.Sprintf("Your %s was lifted: %s", d.Type.Name, d.Title), ptypes.AlertTypeSuccess)
	}

	updated, err := loadDisciplinary(ctx, id)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, updated), nil
}

var notifyDisciplinary = func(userID, title, message string, alertType ptypes.AlertType) {
	if err := notifications.PushNotification(userID, ptypes.Alert{
		Type:     alertType,
		Title:    title,
		Message:  message,
		URL:      pgtype.Text{String: strings.TrimRight(staffPanelURL(), "/") + "/account", Valid: staffPanelURL() != ""},
		Category: ptypes.AlertCategoryStaff,
	}); err != nil {
		state.Logger.Warn("Failed to notify staff member about a disciplinary", zap.Error(err), zap.String("user_id", userID))
	}
}

func staffPanelURL() string {
	for _, origin := range state.Config.Arcadia.Panel.AllowedOrigins {
		if origin != "" {
			return origin
		}
	}

	return ""
}
