package impls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrIdentityExpired  = errors.New("identityExpired")
	ErrSessionNotActive = errors.New("sessionNotActive")
)

func CheckAuthInsecure(ctx context.Context, token string) (types.AuthData, error) {
	// Sliding expiration: an active session's clock is last_seen_at, bumped
	// to NOW() on every successful CheckAuth call below — so this prunes
	// sessions that have been idle for an hour, not sessions that are simply
	// over an hour old. created_at is untouched and still means "when this
	// row was first created."
	_, err := state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE last_seen_at < NOW() - INTERVAL '1 hour'")

	if err != nil {
		return types.AuthData{}, err
	}

	_, err = state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE state = 'pending' AND created_at < NOW() - INTERVAL '5 minutes'")

	if err != nil {
		return types.AuthData{}, err
	}

	var count int64

	err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM staffpanel__authchain WHERE token = $1", token).Scan(&count)

	if err != nil {
		return types.AuthData{}, err
	}

	if count == 0 {
		return types.AuthData{}, ErrIdentityExpired
	}

	var (
		userID    string
		createdAt time.Time
		sessState string
	)

	err = state.Pool.QueryRow(ctx, "SELECT user_id, created_at, state FROM staffpanel__authchain WHERE token = $1", token).Scan(&userID, &createdAt, &sessState)

	if err != nil {
		return types.AuthData{}, err
	}

	var (
		positions []pgtype.UUID
		isBot     bool
	)

	err = state.Pool.QueryRow(ctx, `
		SELECT sm.positions, COALESCE(iuc.bot, false)
		FROM staff_members sm
		LEFT JOIN internal_user_cache__discord iuc ON iuc.id = sm.user_id
		WHERE sm.user_id = $1`, userID).Scan(&positions, &isBot)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.AuthData{}, ErrIdentityExpired
		}

		return types.AuthData{}, err
	}

	if len(positions) == 0 {
		return types.AuthData{}, ErrIdentityExpired
	}

	if isBot {
		return types.AuthData{}, ErrIdentityExpired
	}

	return types.AuthData{
		UserID:    userID,
		CreatedAt: createdAt.Unix(),
		State:     sessState,
	}, nil
}

func CheckAuth(ctx context.Context, token string) (types.AuthData, error) {
	data, err := CheckAuthInsecure(ctx, token)

	if err != nil {
		return types.AuthData{}, err
	}

	if data.State != "active" {
		return types.AuthData{}, ErrSessionNotActive
	}

	// Sliding expiration: every successful authenticated call for an active
	// session resets its idle clock, so a session in continuous use never
	// hits the 1-hour prune above.
	if _, err := state.Pool.Exec(ctx, "UPDATE staffpanel__authchain SET last_seen_at = NOW() WHERE token = $1", token); err != nil {
		return types.AuthData{}, err
	}

	return data, nil
}

type disciplinaryRow struct {
	ID          pgtype.UUID `db:"id"`
	CreatedAt   time.Time   `db:"created_at"`
	Expiry      *float64    `db:"expiry"`
	Title       string      `db:"title"`
	Description string      `db:"description"`
	Type        string      `db:"type"`

	TypeName           *string    `db:"type_name"`
	TypeDescription    *string    `db:"type_description"`
	TypeSelfAssignable *bool      `db:"self_assignable"`
	TypePermLimits     []string   `db:"perm_limits"`
	TypeAdditory       *bool      `db:"additory"`
	TypeNeedsApproval  *bool      `db:"needs_approval"`
	TypeMaxExpiry      *float64   `db:"max_expiry"`
	TypeCreatedAt      *time.Time `db:"type_created_at"`
}

const disciplinaryQuery = `SELECT d.id, d.created_at, EXTRACT(epoch FROM d.expiry) AS expiry, d.title, d.description, d.type,
        t.name AS type_name, t.description AS type_description, t.self_assignable, t.perm_limits, t.additory, t.needs_approval,
        EXTRACT(epoch FROM t.max_expiry) AS max_expiry, t.created_at AS type_created_at
        FROM staff_disciplinary d LEFT JOIN staff_disciplinary_types t ON t.id = d.type
        WHERE d.user_id = $1`

func GetStaffDisciplinaries(ctx context.Context, userID string, active bool) ([]types.StaffDisciplinary, error) {
	query := disciplinaryQuery

	if active {
		query += " AND NOW() - d.created_at < d.expiry"
	}

	rows, err := state.Pool.Query(ctx, query, userID)

	if err != nil {
		return nil, err
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[disciplinaryRow])

	if err != nil {
		return nil, err
	}

	disciplinaries := make([]types.StaffDisciplinary, 0, len(records))

	for _, rec := range records {
		if rec.TypeName == nil {
			return nil, pgx.ErrNoRows
		}

		var expiresAt *int64

		if rec.Expiry != nil {
			seconds := int64(*rec.Expiry)
			expiresAt = &seconds
		}

		disciplinaries = append(disciplinaries, types.StaffDisciplinary{
			ID:          UUIDString(rec.ID),
			UserID:      userID,
			CreatedAt:   types.NewTimestamp(rec.CreatedAt),
			ExpiresAt:   expiresAt,
			Title:       rec.Title,
			Description: rec.Description,
			Type: types.StaffDisciplinaryType{
				ID:             rec.Type,
				Name:           *rec.TypeName,
				Description:    derefOr(rec.TypeDescription, ""),
				SelfAssignable: derefOr(rec.TypeSelfAssignable, false),
				PermLimits:     types.NonNilStrings(rec.TypePermLimits),
				Additory:       derefOr(rec.TypeAdditory, false),
				NeedsApproval:  derefOr(rec.TypeNeedsApproval, false),
				MaxExpiry:      rec.TypeMaxExpiry,
				CreatedAt:      types.NewTimestamp(derefOr(rec.TypeCreatedAt, rec.CreatedAt)),
			},
		})
	}

	return disciplinaries, nil
}

func derefOr[T any](v *T, fallback T) T {
	if v == nil {
		return fallback
	}

	return *v
}

type staffPositionRow struct {
	ID                 pgtype.UUID  `db:"id"`
	Name               string       `db:"name"`
	RoleID             string       `db:"role_id"`
	Perms              []string     `db:"perms"`
	CorrespondingRoles []types.Link `db:"corresponding_roles"`
	Icon               string       `db:"icon"`
	Index              int32        `db:"index"`
	CreatedAt          time.Time    `db:"created_at"`
}

func GetStaffMember(ctx context.Context, userID string) (types.StaffMember, error) {
	var (
		positionIDs   []pgtype.UUID
		permOverrides []string
		noAutosync    bool
		unaccounted   bool
		mfaVerified   bool
		createdAt     time.Time
	)

	err := state.Pool.QueryRow(ctx,
		"SELECT positions, perm_overrides, no_autosync, unaccounted, mfa_verified, created_at FROM staff_members WHERE user_id = $1",
		userID,
	).Scan(&positionIDs, &permOverrides, &noAutosync, &unaccounted, &mfaVerified, &createdAt)

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting staff perms of user %s: %s", userID, err)
	}

	rows, err := state.Pool.Query(ctx,
		"SELECT id, name, role_id, perms, corresponding_roles, icon, index, created_at FROM staff_positions WHERE id = ANY($1)",
		positionIDs,
	)

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting positions of user %s: %s", userID, err)
	}

	positionRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[staffPositionRow])

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting positions of user %s: %s", userID, err)
	}

	user, err := GetPlatformUser(ctx, userID)

	if err != nil {
		return types.StaffMember{}, err
	}

	grants := perms.StaffGrants{
		Roles:       make([]perms.Role, 0, len(positionRows)),
		Extras:      perms.ParseStrings(permOverrides),
		ConfigOwner: perms.IsConfigOwner(userID),
		BotAccount:  user.Bot,
	}

	positions := make([]types.StaffPosition, 0, len(positionRows))

	for _, p := range positionRows {
		id := UUIDString(p.ID)

		grants.Roles = append(grants.Roles, perms.Role{
			ID:    id,
			Name:  p.Name,
			Index: p.Index,
			Perms: perms.ParseStrings(p.Perms),
		})

		positions = append(positions, types.StaffPosition{
			ID:                 id,
			Name:               p.Name,
			RoleID:             p.RoleID,
			Perms:              types.NonNilStrings(p.Perms),
			CorrespondingRoles: types.NonNilLinks(p.CorrespondingRoles),
			Icon:               p.Icon,
			Index:              p.Index,
			CreatedAt:          types.NewTimestamp(p.CreatedAt),
		})
	}

	disciplinaries, err := GetStaffDisciplinaries(ctx, userID, true)

	if err != nil {
		return types.StaffMember{}, err
	}

	resolved := resolveWithDisciplinaries(grants, disciplinaries)

	return types.StaffMember{
		UserID:         userID,
		User:           user,
		Positions:      positions,
		Disciplinaries: disciplinaries,
		PermOverrides:  types.NonNilStrings(permOverrides),
		ResolvedPerms:  resolved.Strings(),
		NoAutosync:     noAutosync,
		Unaccounted:    unaccounted,
		MfaVerified:    mfaVerified,
		CreatedAt:      types.NewTimestamp(createdAt),
		Rank:           grants.Rank(),
		Grants:         grants,
	}, nil
}

func resolveWithDisciplinaries(grants perms.StaffGrants, disciplinaries []types.StaffDisciplinary) perms.Set {
	if grants.BotAccount {
		return perms.Staff.NewSet()
	}

	if len(disciplinaries) == 0 || grants.ConfigOwner {
		return grants.Resolve()
	}

	var (
		resolved = grants.Resolve()
		applied  = perms.Staff.NewSet()
	)

	for _, disc := range disciplinaries {
		limits := perms.ParseStrings(disc.Type.PermLimits)

		if !disc.Type.Additory {
			resolved = applied
		}

		resolved = resolved.With(limits...)
		applied = applied.With(limits...)
	}

	return resolved
}
