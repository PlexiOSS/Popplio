// Package assets implements the data request/deletion tasks run for a user.
//
// A task walks every table holding data about the user, following the
// references between them, and either exports or deletes what it finds.
// Progress is written to a task log the user can poll while it runs.
package assets

import (
	"fmt"

	"github.com/PlexiOSS/Keel/uuidutil"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// defaultFetcher and defaultDeleter are intentionally left as raw pgx, not
// converted to sqlc: tableName/columnName here are discovered at runtime by
// getAllTableRefs walking pg_constraint (see tablerefs.go) — every FK in the
// schema, not a fixed enum — so the query shape genuinely can't be
// represented as a static sqlc query, the same reasoning that keeps
// list.search_list on raw pgx.
func defaultFetcher(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) ([]map[string]any, error) {
	l.Info("Fetching table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	rows, err := tx.Query(state.Context, "SELECT * FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return nil, fmt.Errorf("failed to query table: %w", err)
	}

	return pgx.CollectRows(rows, pgx.RowToMap)
}

func defaultDeleter(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) error {
	l.Info("Deleting from table", zap.String("table", tableName), zap.String("column", columnName), zap.String("id", id))
	_, err := tx.Exec(state.Context, "DELETE FROM "+tableName+" WHERE "+columnName+" = $1", id)

	if err != nil {
		return fmt.Errorf("failed to delete from table: %w", err)
	}

	return nil
}

// Table Logic represents a list of foreign keys for a table
type TableLogic struct {
	// GetIdsForUser returns a list of IDs for the user that should be fetched/deleted
	GetIdsForUser func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error)

	// Fetch returns a list of rows for the given table, column and ID
	Fetch func(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) ([]map[string]any, error)

	// Delete deletes all rows for the given table, column and ID
	Delete func(tx pgx.Tx, l *zap.Logger, tableName, columnName, id string) error
}

// Table specific converters
//
// This is run on the table name, not the foreign key
type TableTransformers struct {
	// Convert converts data to the final data for a fetch. This is called on both 'data_request' and 'data_delete'
	//
	// This is useful when having secrets in the database that should not be exposed
	Fetch []func(data []map[string]any) ([]map[string]any, error)
}

var tableLogic = map[string]TableLogic{
	"teams": {
		GetIdsForUser: func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error) {
			// Select all team IDs
			var teamIds []string

			rows, err := db.New(tx).GetUserTeamMembershipsForDataTask(state.Context, id)

			if err != nil {
				return nil, fmt.Errorf("failed to query table: %w", err)
			}

			for _, row := range rows {
				teamId := uuid.UUID(row.TeamID.Bytes).String()

				if row.DataHolder {
					teamIds = append(teamIds, teamId)
				} else {
					if perms.Entity.ResolveStrings(row.Flags).Has(perms.EntityOwner) {
						teamIds = append(teamIds, teamId)
					} else {
						l.Warn("User does not have permission to dump team [not an owner and not data_holder]", zap.String("team_id", teamId), zap.String("user_id", id))
					}
				}
			}

			return teamIds, nil
		},
		Fetch:  defaultFetcher,
		Delete: defaultDeleter,
	},
	"users": {
		GetIdsForUser: func(tx pgx.Tx, l *zap.Logger, id string) ([]string, error) {
			return []string{id}, nil
		},
		Fetch:  defaultFetcher,
		Delete: defaultDeleter,
	},
}

var tableTransformer = map[string]TableTransformers{
	"staffpanel__authchain": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				for i := range data {
					// Redact tokens to avoid leaking them
					data[i]["token"] = "REDACTED"
					data[i]["popplio_token"] = "REDACTED"
				}

				return data, nil
			},
		},
	},
	"staffpanel__paneldata": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				for i := range data {
					// Redact MFA tokens to avoid leaking them
					data[i]["mfa_secret"] = "REDACTED"
					data[i]["mfa_verified"] = "REDACTED"
				}

				return data, nil
			},
		},
	},
	"bots": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				for i := range data {
					// Redact tokens to avoid leaking them but only if theyre not in a team
					if data[i]["team_owner"] != nil {
						data[i]["api_token"] = "REDACTED_AS_IN_A_TEAM"
					}
				}

				return data, nil
			},
		},
	},
	"*": {
		Fetch: []func(data []map[string]any) ([]map[string]any, error){
			func(data []map[string]any) ([]map[string]any, error) {
				var uuidKeys []string
				for i := range data {
					// Format [16]byte as a UUID as this is always true for our purposes
					if len(uuidKeys) == 0 {
						for key := range data[i] {
							if k, ok := data[i][key].([16]byte); ok {
								data[i][key] = uuidutil.Encode(k)
								uuidKeys = append(uuidKeys, key)
							}
						}
					} else {
						// Fast path
						for _, key := range uuidKeys {
							if k, ok := data[i][key].([16]byte); ok {
								data[i][key] = uuidutil.Encode(k)
							}
						}
					}
				}

				return data, nil
			},
		},
	},
}
