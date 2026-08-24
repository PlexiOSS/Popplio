package moderation

import (
	"context"
	"fmt"
	"strings"

	"popplio/state"
)

const SystemReporterID = "system:moderation"

func FileAutoReport(ctx context.Context, targetType, targetID string, categories []string) error {
	description := fmt.Sprintf(
		"Automatically filed: the submitted description was flagged by OpenAI's moderation endpoint for: %s.",
		strings.Join(categories, ", "),
	)

	_, err := state.Pool.Exec(ctx,
		"INSERT INTO reports (target_type, target_id, reporter_id, reason, description) VALUES ($1, $2, $3, 'tos_violation', $4) ON CONFLICT DO NOTHING",
		targetType, targetID, SystemReporterID, description,
	)

	return err
}
