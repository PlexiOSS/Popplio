// Copyright (C) 2026 NodeByte LTD

package perms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/jackc/pgx/v5"

	"github.com/PlexiOSS/Keel/dovewing"
)

const staffQuery = `SELECT
	sm.perm_overrides,
	COALESCE((
		SELECT json_agg(json_build_object('id', sp.id::text, 'name', sp.name, 'index', sp.index, 'perms', sp.perms) ORDER BY sp.index)
		FROM staff_positions sp
		WHERE sp.id = ANY(sm.positions)
	), '[]'::json),
	COALESCE(iuc.bot, false)
FROM staff_members sm
LEFT JOIN internal_user_cache__discord iuc ON iuc.id = sm.user_id
WHERE sm.user_id = $1`

type StaffGrants struct {
	Roles       []Role `json:"roles"`
	Extras      []Perm `json:"extras"`
	ConfigOwner bool   `json:"config_owner"`
	BotAccount  bool   `json:"bot_account"`
}

func (g StaffGrants) Resolve() Set {
	if g.BotAccount {
		return Staff.NewSet()
	}

	set := Staff.Resolve(g.Roles, g.Extras...)

	if g.ConfigOwner {
		return set.With(StaffAdministrator)
	}

	return set
}

func (g StaffGrants) TopRole() (Role, bool) {
	if len(g.Roles) == 0 {
		return Role{}, false
	}

	top := g.Roles[0]

	for _, r := range g.Roles[1:] {
		if r.Index < top.Index {
			top = r
		}
	}

	return top, true
}

func (g StaffGrants) Rank() int32 {
	if g.BotAccount {
		return NoRank
	}

	if g.ConfigOwner {
		return OwnerRank
	}

	top, ok := g.TopRole()

	if !ok {
		return NoRank
	}

	return top.Index
}

const NoRank int32 = 1<<31 - 1

type staffRoleJSON struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Index int32    `json:"index"`
	Perms []string `json:"perms"`
}

func LoadStaff(ctx context.Context, userID string) (StaffGrants, error) {
	var (
		extras []string
		roles  []byte
		bot    bool
	)

	owner := IsConfigOwner(userID)

	err := state.Pool.QueryRow(ctx, staffQuery, userID).Scan(&extras, &roles, &bot)

	if err != nil {
		if owner && errors.Is(err, pgx.ErrNoRows) {
			return StaffGrants{ConfigOwner: true, BotAccount: cachedBotFlag(ctx, userID)}, nil
		}

		return StaffGrants{}, fmt.Errorf("error while getting staff perms of user %s: %w", userID, err)
	}

	var rows []staffRoleJSON

	if err := json.Unmarshal(roles, &rows); err != nil {
		return StaffGrants{}, fmt.Errorf("error while getting staff roles of user %s: %w", userID, err)
	}

	g := StaffGrants{
		Roles:       make([]Role, 0, len(rows)),
		Extras:      ParseStrings(extras),
		ConfigOwner: owner,
		BotAccount:  bot,
	}

	for _, row := range rows {
		g.Roles = append(g.Roles, Role{
			ID:    row.ID,
			Name:  row.Name,
			Index: row.Index,
			Perms: ParseStrings(row.Perms),
		})
	}

	return g, nil
}

func StaffPerms(ctx context.Context, userID string) (Set, error) {
	g, err := LoadStaff(ctx, userID)

	if err != nil {
		return Set{}, err
	}

	return g.Resolve(), nil
}

type InternalUserCacheDiscordRow struct {
	ID       string `db:"id"`
	Username string `db:"username"`
	Bot      bool   `db:"bot"`
}

func cachedBotFlag(ctx context.Context, userID string) bool {
	var bot bool

	err := state.Pool.QueryRow(ctx, "SELECT bot FROM internal_user_cache__discord WHERE id = $1", userID).Scan(&bot)

	if err != nil {
		return false
	}

	return bot
}

var ErrBotAccount = errors.New("bot accounts cannot hold staff permissions")

func RejectBotAccount(ctx context.Context, userID string) error {
	user, err := dovewing.GetUser(ctx, userID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("could not check whether %s is a bot: %w", userID, err)
	}

	if user.Bot {
		return ErrBotAccount
	}

	return nil
}
