// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"popplio/arcadia/types"
	"popplio/badgerules"
	"popplio/bgtasks"
	"popplio/db"
	"popplio/linkedroles"
	"popplio/perms"
	"popplio/state"

	"go.uber.org/zap"
)

var linkedRoleKey = regexp.MustCompile(`^[a-z0-9_]{1,50}$`)

func ruleToMap(r *badgerules.Rule) map[string]any {
	if r == nil {
		return nil
	}

	raw, _ := json.Marshal(r)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)

	return m
}

func mapToRule(m map[string]any) *badgerules.Rule {
	if m == nil {
		return nil
	}

	raw, _ := json.Marshal(m)
	r := &badgerules.Rule{}

	if err := json.Unmarshal(raw, r); err != nil || r.Type == "" {
		return nil
	}

	return r
}

func validateBadgeUpsert(ctx context.Context, a *types.BadgeUpsert) string {
	if err := badgerules.Validate(a.AutoRule, a.TargetTypes); err != nil {
		return "Automatic rule: " + err.Error()
	}

	if !a.LinkedRole {
		return ""
	}

	userBadge := false

	for _, t := range a.TargetTypes {
		if t == "user" {
			userBadge = true
		}
	}

	if !userBadge {
		return "Only badges that can be given to users can be used for linked roles"
	}

	if !linkedRoleKey.MatchString(a.ID) {
		return "Linked-role badges need an ID of lowercase letters, numbers and underscores (up to 50 characters)"
	}

	others, err := db.New(state.Pool).CountOtherLinkedRoleBadges(ctx, a.ID)

	if err != nil {
		return "Could not check how many badges are used for linked roles"
	}

	if others >= linkedroles.MaxBadges {
		return fmt.Sprintf("Discord allows at most %d linked-role badges. Turn it off on another badge first", linkedroles.MaxBadges)
	}

	return ""
}

func afterBadgeChange() {
	linkedroles.RegisterMetadataAsync()

	go func() {
		if err := bgtasks.BadgeSync(context.Background()); err != nil {
			state.Logger.Warn("Badge sync after a badge change failed", zap.Error(err))
		}
	}()
}

// The badge catalog: what badges exist, their icon/color, and which entity
// types they can go on. Assigning one to an entity is a separate RPC action
// (see rpc.assignBadgeSet), not a panel op — this only manages the catalog.

func (s *Server) updateBadges(ctx context.Context, q *types.QUpdateBadges) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		badgeRows, err := db.New(state.Pool).ListBadges(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		badges := make([]types.Badge, 0, len(badgeRows))

		for _, b := range badgeRows {
			badges = append(badges, types.Badge{
				ID:          b.ID,
				Name:        b.Name,
				Description: b.Description,
				Icon:        b.Icon,
				Color:       b.Color,
				TargetTypes: types.NonNilStrings(b.TargetTypes),
				CreatedAt:   types.NewTimestamp(b.CreatedAt.Time),
				CreatedBy:   b.CreatedBy,
				LastUpdated: types.NewTimestamp(b.LastUpdated.Time),
				UpdatedBy:   b.UpdatedBy,
				AutoRule:    mapToRule(b.AutoRule),
				LinkedRole:  b.LinkedRole,
			})
		}

		return writeJSON(http.StatusOK, badges), nil
	case q.Action.RuleKinds != nil:
		return writeJSON(http.StatusOK, badgerules.Kinds), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to create badges [manage_badges]"), nil
		}

		if msg := validateBadgeUpsert(ctx, action); msg != "" {
			return writeText(http.StatusBadRequest, msg), nil
		}

		err := db.New(state.Pool).InsertBadge(ctx, db.InsertBadgeParams{
			ID:          action.ID,
			Name:        action.Name,
			Description: action.Description,
			Icon:        action.Icon,
			Color:       action.Color,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			CreatedBy:   authData.UserID,
			AutoRule:    ruleToMap(action.AutoRule),
			LinkedRole:  action.LinkedRole,
		})

		if err != nil {
			return response{}, newError(err)
		}

		afterBadgeChange()

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to update badges [manage_badges]"), nil
		}

		if msg := validateBadgeUpsert(ctx, action); msg != "" {
			return writeText(http.StatusBadRequest, msg), nil
		}

		exists, err := db.New(state.Pool).CountBadgeByID(ctx, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		err = db.New(state.Pool).UpdateBadge(ctx, db.UpdateBadgeParams{
			Name:        action.Name,
			Description: action.Description,
			Icon:        action.Icon,
			Color:       action.Color,
			TargetTypes: types.NonNilStrings(action.TargetTypes),
			UpdatedBy:   authData.UserID,
			ID:          action.ID,
			AutoRule:    ruleToMap(action.AutoRule),
			LinkedRole:  action.LinkedRole,
		})

		if err != nil {
			return response{}, newError(err)
		}

		afterBadgeChange()

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageBadges) {
			return writeText(http.StatusForbidden, "You do not have permission to delete badges [manage_badges]"), nil
		}

		id := q.Action.Delete.ID

		exists, err := db.New(state.Pool).CountBadgeByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if resp := requireExists(exists); resp != nil {
			return *resp, nil
		}

		if err := db.New(state.Pool).DeleteBadge(ctx, id); err != nil {
			return response{}, newError(err)
		}

		afterBadgeChange()

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No badge action was specified")
	}
}
