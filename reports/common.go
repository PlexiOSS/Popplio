// Package reports implements the generic content-report system: any user
// may report any votable entity (bots, servers, packs, ...), keyed the same
// (target_type, target_id) way popplio/votes already keys entity_votes.
// Reports are reviewed by staff in Arcadia (see arcadia/panel/ops_reports.go)
// — there is deliberately no public listing/review route, matching how
// Blog/Partners never got one either.
package reports

import (
	"context"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/jackc/pgx/v5"
)

type DbConn interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TargetInfo is what's needed to show a report in the staff panel: a
// human-readable name and a link to the reported entity.
type TargetInfo struct {
	Name string
	URL  string
}

// GetTargetInfo confirms a report target actually exists and returns
// enough to display it. Unlike votes.GetEntityInfo, this deliberately does
// NOT reject vote-banned/already-flagged entities — you should still be
// able to report something that's already banned, e.g. to add detail or
// flag a fresh violation.
func GetTargetInfo(ctx context.Context, c DbConn, targetType, targetId string) (*TargetInfo, error) {
	frontend := state.Config.Sites.Frontend

	switch targetType {
	case "bot":
		var exists bool
		err := c.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bots WHERE bot_id = $1)", targetId).Scan(&exists)

		if err != nil {
			return nil, fmt.Errorf("failed to check bot exists: %w", err)
		}

		if !exists {
			return nil, errors.New("bot not found")
		}

		return &TargetInfo{Name: targetId, URL: frontend + "/bots/" + targetId}, nil
	case "server":
		var name string
		err := c.QueryRow(ctx, "SELECT name FROM servers WHERE server_id = $1", targetId).Scan(&name)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch server: %w", err)
		}

		return &TargetInfo{Name: name, URL: frontend + "/servers/" + targetId}, nil
	case "pack":
		var name string
		err := c.QueryRow(ctx, "SELECT name FROM packs WHERE url = $1", targetId).Scan(&name)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("pack not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch pack: %w", err)
		}

		return &TargetInfo{Name: name, URL: frontend + "/packs/" + targetId}, nil
	case "team":
		var name string
		err := c.QueryRow(ctx, "SELECT name FROM teams WHERE id = $1", targetId).Scan(&name)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("team not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch team: %w", err)
		}

		return &TargetInfo{Name: name, URL: frontend + "/teams/" + targetId}, nil
	default:
		return nil, errors.New("unsupported report target type: " + targetType)
	}
}
