// Copyright (C) 2026 NodeByte LTD

package pagestyle

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var Gradients = []string{"aurora", "sunset", "ocean", "forest", "candy", "midnight", "ember", "mono"}

var (
	ErrNotPremium  = errors.New("page styles need premium or a Page Style pass")
	ErrBadGradient = errors.New("unknown gradient preset")
	ErrBadAccent   = errors.New("accent colour is too dark or too light to read")
	ErrNotFound    = errors.New("listing not found")
)

func ValidGradient(id string) bool {
	for _, g := range Gradients {
		if g == id {
			return true
		}
	}

	return false
}

func channel(hex string) float64 {
	v, _ := strconv.ParseUint(hex, 16, 8)
	c := float64(v) / 255

	if c <= 0.04045 {
		return c / 12.92
	}

	return math.Pow((c+0.055)/1.055, 2.4)
}

func Luminance(color string) float64 {
	color = strings.TrimPrefix(strings.ToLower(color), "#")

	if len(color) != 6 {
		return -1
	}

	return 0.2126*channel(color[0:2]) + 0.7152*channel(color[2:4]) + 0.0722*channel(color[4:6])
}

func contrast(a, b float64) float64 {
	if a < b {
		a, b = b, a
	}

	return (a + 0.05) / (b + 0.05)
}

func ReadableAccent(color string) bool {
	l := Luminance(color)

	if l < 0 {
		return false
	}

	return contrast(l, Luminance("#ffffff")) >= 2.2 && contrast(l, Luminance("#09090b")) >= 2.6
}

func Premium(ctx context.Context, targetType, targetID string) (bool, error) {
	q := db.New(state.Pool)

	var premium bool
	var err error

	switch targetType {
	case "bot":
		row, e := q.GetBotTypePremium(ctx, targetID)
		premium, err = row.Premium, e
	case "server":
		row, e := q.GetServerTypePremium(ctx, targetID)
		premium, err = row.Premium, e
	default:
		return false, ErrNotFound
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}

	if err != nil || premium {
		return premium, err
	}

	until, err := UnlockedUntil(ctx, targetType, targetID)

	return until != nil && until.After(time.Now()), err
}

func UnlockedUntil(ctx context.Context, targetType, targetID string) (*time.Time, error) {
	until, err := db.New(state.Pool).GetPageStyleUnlock(ctx, db.GetPageStyleUnlockParams{TargetType: targetType, TargetID: targetID})

	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !until.Valid) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	t := until.Time
	return &t, nil
}

func text(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}

	s := t.String
	return &s
}

func toStyle(row db.GetPageStyleRow) types.PageStyle {
	return types.PageStyle{
		Gradient:      text(row.Gradient),
		Accent:        text(row.Accent),
		ImageVersion:  text(row.ImageVersion),
		Dim:           int(row.Dim),
		Blur:          int(row.Blur),
		ImagePosition: row.ImagePosition,
		Fixed:         row.Fixed,
	}
}

func defaults() types.PageStyle {
	return types.PageStyle{Dim: 55, ImagePosition: "center"}
}

func Public(ctx context.Context, targetType, targetID string) (*types.PageStyle, error) {
	premium, err := Premium(ctx, targetType, targetID)

	if err != nil || !premium {
		return nil, err
	}

	row, err := db.New(state.Pool).GetPageStyle(ctx, db.GetPageStyleParams{TargetType: targetType, TargetID: targetID})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	style := toStyle(row)

	if style.Gradient == nil && style.Accent == nil && style.ImageVersion == nil {
		return nil, nil
	}

	return &style, nil
}

func Managed(ctx context.Context, targetType, targetID string) (types.ManagedPageStyle, error) {
	premium, err := Premium(ctx, targetType, targetID)

	if err != nil {
		return types.ManagedPageStyle{}, err
	}

	out := types.ManagedPageStyle{PageStyle: defaults(), Premium: premium}

	row, err := db.New(state.Pool).GetPageStyle(ctx, db.GetPageStyleParams{TargetType: targetType, TargetID: targetID})

	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}

	if err != nil {
		return out, err
	}

	out.PageStyle = toStyle(row)
	out.PendingVersion = text(row.PendingImageVersion)
	out.RejectedReason = text(row.RejectedReason)

	if until, err := UnlockedUntil(ctx, targetType, targetID); err != nil {
		return out, err
	} else if until != nil && until.After(time.Now()) {
		out.UnlockedUntil = until
	}

	if row.PendingSince.Valid {
		t := row.PendingSince.Time
		out.PendingSince = &t
	}

	return out, nil
}

func Update(ctx context.Context, targetType, targetID string, u types.PageStyleUpdate) error {
	premium, err := Premium(ctx, targetType, targetID)

	if err != nil {
		return err
	}

	if !premium {
		return ErrNotPremium
	}

	params := db.UpsertPageStyleSettingsParams{
		TargetType:    targetType,
		TargetID:      targetID,
		Dim:           int16(u.Dim),
		Blur:          int16(u.Blur),
		ImagePosition: u.ImagePosition,
		Fixed:         u.Fixed,
	}

	if u.Gradient != nil && *u.Gradient != "" {
		if !ValidGradient(*u.Gradient) {
			return ErrBadGradient
		}

		params.Gradient = pgtype.Text{String: *u.Gradient, Valid: true}
	}

	if u.Accent != nil && *u.Accent != "" {
		accent := strings.ToLower(*u.Accent)

		if !ReadableAccent(accent) {
			return ErrBadAccent
		}

		params.Accent = pgtype.Text{String: accent, Valid: true}
	}

	q := db.New(state.Pool)

	if err := q.UpsertPageStyleSettings(ctx, params); err != nil {
		return err
	}

	if u.RemoveImage {
		return q.RemovePageImage(ctx, db.RemovePageImageParams{TargetType: targetType, TargetID: targetID})
	}

	return nil
}

func SubmitImage(ctx context.Context, targetType, targetID, version string) error {
	premium, err := Premium(ctx, targetType, targetID)

	if err != nil {
		return err
	}

	if !premium {
		return ErrNotPremium
	}

	return db.New(state.Pool).SetPendingPageImage(ctx, db.SetPendingPageImageParams{
		TargetType: targetType,
		TargetID:   targetID,
		Version:    pgtype.Text{String: strings.ToLower(version), Valid: true},
	})
}
