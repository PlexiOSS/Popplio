package seedci

import (
	"encoding/json"
	"fmt"
	"kitehelper/common"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type column struct {
	TableName  string          `json:"table_name"`
	ColumnName string          `json:"column_name"`
	Type       string          `json:"type"`
	Nullable   bool            `json:"nullable"`
	Array      bool            `json:"array"`
	DefaultSQL *string         `json:"default_sql"`
	DefaultVal json.RawMessage `json:"default_val"`
	Secret     bool            `json:"secret"`
}

var (
	tablePattern   = regexp.MustCompile(`(?s)CREATE TABLE public\.(\w+) \((.*?)\n\);`)
	columnPattern  = regexp.MustCompile(`^"?(\w+)"?\s+(.+)$`)
	defaultPattern = regexp.MustCompile(`DEFAULT (.+?)(?: NOT NULL)?$`)
	typeSplit      = regexp.MustCompile(` DEFAULT | NOT NULL`)
	skipTables     = []string{"goose_db_version"}
	secretColumns  = []string{"bots.api_token", "servers.invite", "users.api_token"}
)

func Write() (string, int, error) {
	root := common.GetRepoRoot()
	schema, err := os.ReadFile(filepath.Join(root, "db", "schema.sql"))

	if err != nil {
		return "", 0, fmt.Errorf("reading db/schema.sql: %w", err)
	}

	cols := build(string(schema))
	encoded, err := json.Marshal(cols)

	if err != nil {
		return "", 0, fmt.Errorf("encoding seed: %w", err)
	}

	seedPath := filepath.Join(root, "data", "seed-ci.json")

	if err := os.WriteFile(seedPath, encoded, 0o644); err != nil {
		return "", 0, fmt.Errorf("writing %s: %w", seedPath, err)
	}

	return seedPath, len(cols), nil
}

func SeedCI(progname string, args []string) {
	path, n, err := Write()

	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s from db/schema.sql: %d columns\n", path, n)
}

func build(schema string) []column {
	var next []column

	for _, m := range tablePattern.FindAllStringSubmatch(schema, -1) {
		table := m[1]

		if slices.Contains(skipTables, table) {
			continue
		}

		for _, line := range strings.Split(m[2], "\n") {
			line = strings.TrimSuffix(strings.TrimSpace(line), ",")

			if line == "" || strings.HasPrefix(line, "CONSTRAINT") {
				continue
			}

			cm := columnPattern.FindStringSubmatch(line)

			if cm == nil {
				continue
			}

			name, rest := cm[1], cm[2]
			typ := strings.TrimSpace(typeSplit.Split(rest, 2)[0])
			c := column{
				DefaultVal: json.RawMessage("null"),
				TableName:  table,
				ColumnName: name,
				Type:       strings.ReplaceAll(typ, "[]", ""),
				Nullable:   !strings.Contains(rest, "NOT NULL"),
				Array:      strings.Contains(typ, "[]"),
				Secret:     slices.Contains(secretColumns, table+"."+name),
			}

			if dm := defaultPattern.FindStringSubmatch(rest); dm != nil {
				def := dm[1]
				c.DefaultSQL = &def
			}

			next = append(next, c)
		}
	}

	return next
}
