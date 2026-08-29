package impls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// numericToFloat64Ptr converts a nullable pgtype.Numeric (as returned by an
// EXTRACT(epoch FROM ...) expression) to the *float64 this package's structs
// use, mirroring the nil-on-NULL behavior the raw driver gave for free.
func numericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}

	f, err := n.Float64Value()

	if err != nil || !f.Valid {
		return nil
	}

	return &f.Float64
}

var (
	ErrIdentityExpired  = errors.New("identityExpired")
	ErrSessionNotActive = errors.New("sessionNotActive")
)

type StaffAuthChainRow struct {
	Token      string    `db:"token"`
	UserID     string    `db:"user_id"`
	CreatedAt  time.Time `db:"created_at"`
	State      string    `db:"state"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

func CheckAuthInsecure(ctx context.Context, token string) (types.AuthData, error) {
	q := db.New(state.Pool)

	err := q.DeleteStaleAuthChainEntries(ctx)

	if err != nil {
		return types.AuthData{}, err
	}

	err = q.DeleteExpiredPendingAuthChainEntries(ctx)

	if err != nil {
		return types.AuthData{}, err
	}

	count, err := q.CountAuthChainByToken(ctx, token)

	if err != nil {
		return types.AuthData{}, err
	}

	if count == 0 {
		return types.AuthData{}, ErrIdentityExpired
	}

	chain, err := q.GetAuthChainByToken(ctx, token)

	if err != nil {
		return types.AuthData{}, err
	}

	userID, createdAt, sessState := chain.UserID, chain.CreatedAt.Time, chain.State

	row, err := q.GetStaffPositionsAndBotFlag(ctx, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.AuthData{}, ErrIdentityExpired
		}

		return types.AuthData{}, err
	}

	positions, isBot := row.Positions, row.Bot

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

	if err := db.New(state.Pool).TouchAuthChainSeen(ctx, token); err != nil {
		return types.AuthData{}, err
	}

	return data, nil
}

func GetStaffDisciplinaries(ctx context.Context, userID string, active bool) ([]types.StaffDisciplinary, error) {
	q := db.New(state.Pool)

	disciplinaries := make([]types.StaffDisciplinary, 0)

	if active {
		rows, err := q.GetActiveStaffDisciplinaries(ctx, userID)

		if err != nil {
			return nil, err
		}

		for _, rec := range rows {
			d, err := disciplinaryFromActiveRow(rec, userID)

			if err != nil {
				return nil, err
			}

			disciplinaries = append(disciplinaries, d)
		}

		return disciplinaries, nil
	}

	rows, err := q.GetStaffDisciplinaries(ctx, userID)

	if err != nil {
		return nil, err
	}

	for _, rec := range rows {
		d, err := disciplinaryFromRow(rec, userID)

		if err != nil {
			return nil, err
		}

		disciplinaries = append(disciplinaries, d)
	}

	return disciplinaries, nil
}

func disciplinaryFromRow(rec db.GetStaffDisciplinariesRow, userID string) (types.StaffDisciplinary, error) {
	if !rec.TypeName.Valid {
		return types.StaffDisciplinary{}, pgx.ErrNoRows
	}

	var typeCreatedAt time.Time
	if rec.TypeCreatedAt.Valid {
		typeCreatedAt = rec.TypeCreatedAt.Time
	} else {
		typeCreatedAt = rec.CreatedAt.Time
	}

	return types.StaffDisciplinary{
		ID:          UUIDString(rec.ID),
		UserID:      userID,
		CreatedAt:   types.NewTimestamp(rec.CreatedAt.Time),
		ExpiresAt:   epochFloatToSeconds(numericToFloat64Ptr(rec.Expiry)),
		Title:       rec.Title,
		Description: rec.Description,
		Type: types.StaffDisciplinaryType{
			ID:             rec.Type,
			Name:           rec.TypeName.String,
			Description:    rec.TypeDescription.String,
			SelfAssignable: rec.SelfAssignable.Bool,
			PermLimits:     types.NonNilStrings(rec.PermLimits),
			Additory:       rec.Additory.Bool,
			NeedsApproval:  rec.NeedsApproval.Bool,
			MaxExpiry:      numericToFloat64Ptr(rec.MaxExpiry),
			CreatedAt:      types.NewTimestamp(typeCreatedAt),
		},
	}, nil
}

func disciplinaryFromActiveRow(rec db.GetActiveStaffDisciplinariesRow, userID string) (types.StaffDisciplinary, error) {
	// Same shape as GetStaffDisciplinariesRow, just from the filtered query
	// -- converting through GetStaffDisciplinariesRow keeps the mapping in
	// one place.
	return disciplinaryFromRow(db.GetStaffDisciplinariesRow(rec), userID)
}

func epochFloatToSeconds(f *float64) *int64 {
	if f == nil {
		return nil
	}

	seconds := int64(*f)
	return &seconds
}

type StaffPositionRow struct {
	ID                 pgtype.UUID  `db:"id"`
	Name               string       `db:"name"`
	RoleID             string       `db:"role_id"`
	Perms              []string     `db:"perms"`
	CorrespondingRoles []types.Link `db:"corresponding_roles"`
	Icon               string       `db:"icon"`
	Index              int32        `db:"index"`
	CreatedAt          time.Time    `db:"created_at"`
}

type StaffMemberRow struct {
	UserID        string        `db:"user_id"`
	Positions     []pgtype.UUID `db:"positions"`
	PermOverrides []string      `db:"perm_overrides"`
	NoAutosync    bool          `db:"no_autosync"`
	Unaccounted   bool          `db:"unaccounted"`
	MFAVerified   bool          `db:"mfa_verified"`
	CreatedAt     time.Time     `db:"created_at"`
}

func GetStaffMember(ctx context.Context, userID string) (types.StaffMember, error) {
	q := db.New(state.Pool)

	member, err := q.GetStaffMemberRaw(ctx, userID)

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting staff perms of user %s: %s", userID, err)
	}

	positionIDs, permOverrides, noAutosync, unaccounted, mfaVerified, createdAt :=
		member.Positions, member.PermOverrides, member.NoAutosync, member.Unaccounted, member.MfaVerified, member.CreatedAt.Time

	rawPositions, err := q.GetStaffPositionsByIDs(ctx, positionIDs)

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting positions of user %s: %s", userID, err)
	}

	positionRows := make([]StaffPositionRow, len(rawPositions))
	for i, row := range rawPositions {
		var correspondingRoles []types.Link
		if err := json.Unmarshal(row.CorrespondingRoles, &correspondingRoles); err != nil {
			return types.StaffMember{}, fmt.Errorf("Error while parsing corresponding_roles of position %s: %s", UUIDString(row.ID), err)
		}

		positionRows[i] = StaffPositionRow{
			ID:                 row.ID,
			Name:               row.Name,
			RoleID:             row.RoleID,
			Perms:              row.Perms,
			CorrespondingRoles: correspondingRoles,
			Icon:               row.Icon,
			Index:              row.Index,
			CreatedAt:          row.CreatedAt.Time,
		}
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
