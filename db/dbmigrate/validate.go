package dbmigrate

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/PlexiOSS/Keel/dbutil"
	"github.com/pressly/goose/v3"
)

type Finding struct {
	Table  string
	Kind   string // "missing_table" | "missing_column"
	Detail string
}

func (f Finding) String() string {
	switch f.Kind {
	case "missing_table":
		return fmt.Sprintf("%s: table does not exist", f.Table)
	case "missing_column":
		return fmt.Sprintf("%s: column %q is expected by Go code but doesn't exist", f.Table, f.Detail)
	default:
		return fmt.Sprintf("%s: %s (%s)", f.Table, f.Detail, f.Kind)
	}
}

func Validate(dsn string, schemas map[string]any) ([]Finding, error) {
	db, err := goose.OpenDBWithDriver("postgres", dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	var findings []Finding

	for table, sample := range schemas {
		tableFindings, err := validateTable(db, table, sample)

		if err != nil {
			return nil, fmt.Errorf("failed to validate %s: %w", table, err)
		}

		findings = append(findings, tableFindings...)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Table != findings[j].Table {
			return findings[i].Table < findings[j].Table
		}
		return findings[i].Detail < findings[j].Detail
	})

	return findings, nil
}

func validateTable(db *sql.DB, table string, sample any) ([]Finding, error) {
	var exists bool

	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)

	if err != nil {
		return nil, err
	}

	if !exists {
		return []Finding{{Table: table, Kind: "missing_table"}}, nil
	}

	rows, err := db.Query("SELECT column_name FROM information_schema.columns WHERE table_name = $1", table)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	actual := map[string]bool{}

	for rows.Next() {
		var col string

		if err := rows.Scan(&col); err != nil {
			return nil, err
		}

		actual[col] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var findings []Finding

	for _, col := range dbutil.GetCols(sample) {
		if !actual[col] {
			findings = append(findings, Finding{Table: table, Kind: "missing_column", Detail: col})
		}
	}

	return findings, nil
}
