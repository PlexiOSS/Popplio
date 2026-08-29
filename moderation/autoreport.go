package moderation

import (
	"context"
	"fmt"
	"strings"

	"popplio/db"
	"popplio/state"
)

const SystemReporterID = "system:moderation"

func FileAutoReport(ctx context.Context, targetType, targetID string, categories []string) error {
	description := fmt.Sprintf(
		"Automatically filed: the submitted description was flagged by OpenAI's moderation endpoint for: %s.",
		strings.Join(categories, ", "),
	)

	return db.New(state.Pool).InsertAutoReport(ctx, db.InsertAutoReportParams{
		TargetType: targetType,
		TargetID:   targetID,
		ReporterID: SystemReporterID,
		Description: description,
	})
}
