package reports

import (
	"context"
	"errors"
	"fmt"

	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

type TargetInfo struct {
	Name string
	URL  string
}

func GetTargetInfo(ctx context.Context, c db.DBTX, targetType, targetId string) (*TargetInfo, error) {
	frontend := state.Config.Sites.Frontend
	q := db.New(c)

	switch targetType {
	case "bot":
		exists, err := q.BotExists(ctx, targetId)

		if err != nil {
			return nil, fmt.Errorf("failed to check bot exists: %w", err)
		}

		if !exists {
			return nil, errors.New("bot not found")
		}

		return &TargetInfo{Name: targetId, URL: frontend + "/bots/" + targetId}, nil
	case "server":
		name, err := q.GetServerName(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch server: %w", err)
		}

		return &TargetInfo{Name: name, URL: frontend + "/servers/" + targetId}, nil
	case "pack":
		name, err := q.GetPackName(ctx, targetId)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("pack not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch pack: %w", err)
		}

		return &TargetInfo{Name: name, URL: frontend + "/packs/" + targetId}, nil
	case "team":
		name, err := q.GetTeamName(ctx, targetId)

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
