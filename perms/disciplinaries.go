// Copyright (C) 2026 NodeByte LTD

package perms

import (
	"context"
	"fmt"

	"popplio/state"
)

type DisciplinaryLimit struct {
	PermLimits []string
	Additory   bool
}

const activeDisciplinaryLimitsQuery = `SELECT t.perm_limits, t.additory
FROM staff_disciplinary d JOIN staff_disciplinary_types t ON t.id = d.type
WHERE d.user_id = $1 AND d.state = 'active'
	AND (d.expiry IS NULL OR COALESCE(d.starts_at, d.created_at) + d.expiry > NOW())`

func LoadDisciplinaryLimits(ctx context.Context, userID string) ([]DisciplinaryLimit, error) {
	rows, err := state.Pool.Query(ctx, activeDisciplinaryLimitsQuery, userID)

	if err != nil {
		return nil, fmt.Errorf("error while getting disciplinaries of user %s: %w", userID, err)
	}

	defer rows.Close()

	var out []DisciplinaryLimit

	for rows.Next() {
		var l DisciplinaryLimit

		if err := rows.Scan(&l.PermLimits, &l.Additory); err != nil {
			return nil, err
		}

		out = append(out, l)
	}

	return out, rows.Err()
}

func ApplyDisciplinaries(g StaffGrants, limits []DisciplinaryLimit) Set {
	if g.BotAccount {
		return Staff.NewSet()
	}

	if len(limits) == 0 || g.ConfigOwner {
		return g.Resolve()
	}

	var (
		resolved = g.Resolve()
		applied  = Staff.NewSet()
	)

	for _, l := range limits {
		parsed := ParseStrings(l.PermLimits)

		if !l.Additory {
			resolved = applied
		}

		resolved = resolved.With(parsed...)
		applied = applied.With(parsed...)
	}

	return resolved
}
