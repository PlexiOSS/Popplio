// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"popplio/apps"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	positionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,39}$`)
	questionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)
)

const (
	maxPositionCooldownSecs = 365 * 24 * 3600
	maxPositionQuestions    = 30
	maxPositionTags         = 10
)

func toAppPositionDTO(row db.AppPosition, pending int64) (types.AppPosition, error) {
	var questions []types.AppPositionQuestion

	if err := json.Unmarshal(row.Questions, &questions); err != nil {
		return types.AppPosition{}, err
	}

	out := types.AppPosition{
		ID:                 row.ID,
		Name:               row.Name,
		Info:               row.Info,
		Summary:            row.Summary,
		Tags:               types.NonNilStrings(row.Tags),
		Questions:          questions,
		Hidden:             row.Hidden,
		Closed:             row.Closed,
		RequiresMainServer: row.RequiresMainServer,
		CooldownSecs:       row.CooldownSecs,
		ApprovalNote:       row.ApprovalNote,
		HelpURL:            row.HelpUrl.String,
		HelpLabel:          row.HelpLabel.String,
		PendingApps:        pending,
		UpdatedAt:          types.NewTimestamp(row.UpdatedAt.Time),
	}

	if out.Questions == nil {
		out.Questions = []types.AppPositionQuestion{}
	}

	if row.Behaviour.Valid {
		if b, ok := apps.Behaviours[row.Behaviour.String]; ok {
			out.Behaviour = &types.AppPositionBehaviour{
				ID:              b.ID,
				Name:            b.Name,
				Description:     b.Description,
				LockedQuestions: types.NonNilStrings(b.LockedQuestions),
			}
		}
	}

	return out, nil
}

func normalisePosition(in *types.AppPositionInput, locked []string) string {
	in.Name = strings.TrimSpace(in.Name)
	in.Info = strings.TrimSpace(in.Info)
	in.Summary = strings.Join(strings.Fields(in.Summary), " ")
	in.ApprovalNote = strings.TrimSpace(in.ApprovalNote)
	in.HelpURL = strings.TrimSpace(in.HelpURL)
	in.HelpLabel = strings.TrimSpace(in.HelpLabel)

	if in.Name == "" || len(in.Name) > 80 {
		return "The name must be between 1 and 80 characters"
	}

	if len(in.Info) > 10000 {
		return "The description can be at most 10,000 characters"
	}

	if len(in.Summary) > 200 {
		return "The summary can be at most 200 characters"
	}

	if len(in.ApprovalNote) > 1000 {
		return "The approval note can be at most 1,000 characters"
	}

	if in.CooldownSecs < 0 || in.CooldownSecs > maxPositionCooldownSecs {
		return "The cooldown must be between 0 and 365 days"
	}

	if in.HelpURL != "" {
		if !strings.HasPrefix(in.HelpURL, "/") && !strings.HasPrefix(in.HelpURL, "https://") {
			return "The help link must start with / or https://"
		}

		if in.HelpLabel == "" || len(in.HelpLabel) > 60 {
			return "Give the help link a label of up to 60 characters"
		}
	} else {
		in.HelpLabel = ""
	}

	tags := make([]string, 0, len(in.Tags))

	for _, tag := range in.Tags {
		tag = strings.TrimSpace(tag)

		if tag == "" || slices.Contains(tags, tag) {
			continue
		}

		if len(tag) > 30 {
			return "Tags can be at most 30 characters"
		}

		tags = append(tags, tag)
	}

	if len(tags) > maxPositionTags {
		return fmt.Sprintf("A position can have at most %d tags", maxPositionTags)
	}

	in.Tags = tags

	if len(in.Questions) == 0 || len(in.Questions) > maxPositionQuestions {
		return fmt.Sprintf("A position needs between 1 and %d questions", maxPositionQuestions)
	}

	seen := map[string]bool{}

	for i := range in.Questions {
		q := &in.Questions[i]
		q.ID = strings.TrimSpace(q.ID)
		q.Question = strings.TrimSpace(q.Question)
		q.Paragraph = strings.TrimSpace(q.Paragraph)
		q.Placeholder = strings.TrimSpace(q.Placeholder)

		if !questionIDPattern.MatchString(q.ID) {
			return fmt.Sprintf("Question %d needs an ID of lowercase letters, numbers, - or _", i+1)
		}

		if seen[q.ID] {
			return fmt.Sprintf("Two questions share the ID %q", q.ID)
		}

		seen[q.ID] = true

		if q.Question == "" || len(q.Question) > 200 {
			return fmt.Sprintf("Question %d needs a question of up to 200 characters", i+1)
		}

		if len(q.Paragraph) > 1000 || len(q.Placeholder) > 200 {
			return fmt.Sprintf("Question %d's help text or placeholder is too long", i+1)
		}
	}

	asQuestions := make([]ptypes.Question, len(in.Questions))

	for i, q := range in.Questions {
		asQuestions[i] = ptypes.Question{
			ID:          q.ID,
			Question:    q.Question,
			Paragraph:   q.Paragraph,
			Placeholder: q.Placeholder,
			Short:       q.Short,
			Type:        q.Type,
			Options:     q.Options,
			Optional:    q.Optional,
			MinLength:   q.MinLength,
			MaxLength:   q.MaxLength,
			Flag:        q.Flag,
		}

		if q.ShowIf != nil {
			asQuestions[i].ShowIf = &ptypes.QuestionCondition{QuestionID: q.ShowIf.QuestionID, Values: q.ShowIf.Values}
		}
	}

	if msg := apps.CheckDefinition(asQuestions); msg != "" {
		return msg
	}

	for _, id := range locked {
		if !seen[id] {
			return fmt.Sprintf("This position's built-in behaviour reads the %q question, so it can't be removed or renamed", id)
		}
	}

	return ""
}

func helpText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func (s *Server) updateAppPositions(ctx context.Context, q *types.QUpdateAppPositions) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	queries := db.New(state.Pool)

	if q.Action.List != nil {
		if !userPerms.Has(perms.StaffManageAppPositions) && !userPerms.Has(perms.StaffViewApps) {
			return writeText(http.StatusForbidden, "You do not have permission to view application forms [view_apps]"), nil
		}

		rows, err := queries.ListAppPositions(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.AppPosition, 0, len(rows))

		for _, row := range rows {
			pending, err := queries.CountPendingAppsForPosition(ctx, row.ID)

			if err != nil {
				return response{}, newError(err)
			}

			dto, err := toAppPositionDTO(row, pending)

			if err != nil {
				continue
			}

			out = append(out, dto)
		}

		return writeJSON(http.StatusOK, out), nil
	}

	if !userPerms.Has(perms.StaffManageAppPositions) {
		return writeText(http.StatusForbidden, "You do not have permission to edit application forms [manage_app_positions]"), nil
	}

	defer apps.Invalidate()

	switch {
	case q.Action.Create != nil:
		in := q.Action.Create
		in.ID = strings.TrimSpace(in.ID)

		if !positionIDPattern.MatchString(in.ID) {
			return writeText(http.StatusBadRequest, "The ID must be 2 to 40 lowercase letters, numbers, - or _"), nil
		}

		if msg := normalisePosition(in, nil); msg != "" {
			return writeText(http.StatusBadRequest, msg), nil
		}

		if _, err := queries.GetAppPosition(ctx, in.ID); err == nil {
			return writeText(http.StatusConflict, "A position with that ID already exists"), nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return response{}, newError(err)
		}

		questions, _ := json.Marshal(in.Questions)

		if err := queries.InsertAppPosition(ctx, db.InsertAppPositionParams{
			ID:                 in.ID,
			Name:               in.Name,
			Info:               in.Info,
			Summary:            in.Summary,
			Tags:               in.Tags,
			Questions:          questions,
			Hidden:             in.Hidden,
			Closed:             in.Closed,
			RequiresMainServer: in.RequiresMainServer,
			CooldownSecs:       in.CooldownSecs,
			ApprovalNote:       in.ApprovalNote,
			HelpUrl:            helpText(in.HelpURL),
			HelpLabel:          helpText(in.HelpLabel),
		}); err != nil {
			return response{}, newError(err)
		}

		return s.positionResponse(ctx, in.ID)
	case q.Action.Update != nil:
		in := q.Action.Update

		existing, err := queries.GetAppPosition(ctx, in.ID)

		if errors.Is(err, pgx.ErrNoRows) {
			return writeText(http.StatusNotFound, "Position not found"), nil
		}

		if err != nil {
			return response{}, newError(err)
		}

		var locked []string

		if existing.Behaviour.Valid {
			locked = apps.Behaviours[existing.Behaviour.String].LockedQuestions
		}

		if msg := normalisePosition(in, locked); msg != "" {
			return writeText(http.StatusBadRequest, msg), nil
		}

		questions, _ := json.Marshal(in.Questions)

		if _, err := queries.UpdateAppPosition(ctx, db.UpdateAppPositionParams{
			ID:                 in.ID,
			Name:               in.Name,
			Info:               in.Info,
			Summary:            in.Summary,
			Tags:               in.Tags,
			Questions:          questions,
			Hidden:             in.Hidden,
			Closed:             in.Closed,
			RequiresMainServer: in.RequiresMainServer,
			CooldownSecs:       in.CooldownSecs,
			ApprovalNote:       in.ApprovalNote,
			HelpUrl:            helpText(in.HelpURL),
			HelpLabel:          helpText(in.HelpLabel),
		}); err != nil {
			return response{}, newError(err)
		}

		return s.positionResponse(ctx, in.ID)
	case q.Action.Delete != nil:
		existing, err := queries.GetAppPosition(ctx, q.Action.Delete.ID)

		if errors.Is(err, pgx.ErrNoRows) {
			return writeText(http.StatusNotFound, "Position not found"), nil
		}

		if err != nil {
			return response{}, newError(err)
		}

		if existing.Behaviour.Valid {
			return writeText(http.StatusForbidden, "Built-in positions can't be deleted. Close or hide it instead."), nil
		}

		pending, err := queries.CountPendingAppsForPosition(ctx, existing.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if pending > 0 {
			return writeText(http.StatusConflict, fmt.Sprintf("%d application%s for this position still need a decision. Review them, or close the position instead.", pending, plural(pending))), nil
		}

		if _, err := queries.DeleteAppPosition(ctx, existing.ID); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Reorder != nil:
		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		defer tx.Rollback(ctx)

		txq := db.New(tx)

		for i, id := range q.Action.Reorder.IDs {
			if err := txq.SetAppPositionOrder(ctx, db.SetAppPositionOrderParams{ID: id, SortOrder: int32(i)}); err != nil {
				return response{}, newError(err)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return writeText(http.StatusBadRequest, "No action specified"), nil
	}
}

func (s *Server) positionResponse(ctx context.Context, id string) (response, error) {
	queries := db.New(state.Pool)

	row, err := queries.GetAppPosition(ctx, id)

	if err != nil {
		return response{}, newError(err)
	}

	pending, err := queries.CountPendingAppsForPosition(ctx, id)

	if err != nil {
		return response{}, newError(err)
	}

	dto, err := toAppPositionDTO(row, pending)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, dto), nil
}
