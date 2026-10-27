package seedci

import (
	"bytes"
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
)

func SeedCI(progname string, args []string) {
	check := slices.Contains(args, "--check")
	root := common.GetRepoRoot()
	schemaPath := filepath.Join(root, "db", "schema.sql")
	seedPath := filepath.Join(root, "data", "seed-ci.json")

	schema, err := os.ReadFile(schemaPath)

	if err != nil {
		fmt.Println("ERROR: reading", schemaPath+":", err)
		os.Exit(1)
	}

	current, err := os.ReadFile(seedPath)

	if err != nil {
		fmt.Println("ERROR: reading", seedPath+":", err)
		os.Exit(1)
	}

	var existing []column

	if err := json.Unmarshal(current, &existing); err != nil {
		fmt.Println("ERROR: parsing", seedPath+":", err)
		os.Exit(1)
	}

	next, added, removed := build(string(schema), existing)

	encoded, err := json.Marshal(next)

	if err != nil {
		fmt.Println("ERROR: encoding seed:", err)
		os.Exit(1)
	}

	encoded = bytes.ReplaceAll(encoded, []byte(`<`), []byte("<"))
	encoded = bytes.ReplaceAll(encoded, []byte(`>`), []byte(">"))
	encoded = bytes.ReplaceAll(encoded, []byte(`&`), []byte("&"))

	for _, c := range added {
		fmt.Println("+", c)
	}

	for _, c := range removed {
		fmt.Println("-", c)
	}

	if bytes.Equal(bytes.TrimSpace(current), encoded) {
		fmt.Println("data/seed-ci.json is up to date with db/schema.sql")
		return
	}

	if check {
		fmt.Println("data/seed-ci.json is out of date with db/schema.sql. Run: kitehelper seed-ci")
		os.Exit(1)
	}

	if err := os.WriteFile(seedPath, encoded, 0o644); err != nil {
		fmt.Println("ERROR: writing", seedPath+":", err)
		os.Exit(1)
	}

	fmt.Printf("Updated data/seed-ci.json: %d columns (%d added, %d removed)\n", len(next), len(added), len(removed))
}

func build(schema string, existing []column) (next []column, added, removed []string) {
	known := make(map[string]column, len(existing))

	for _, c := range existing {
		known[c.TableName+"."+c.ColumnName] = c
	}

	schemaTables := map[string]bool{}
	seen := map[string]bool{}

	for _, m := range tablePattern.FindAllStringSubmatch(schema, -1) {
		table := m[1]
		schemaTables[table] = true

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
			key := table + "." + name
			seen[key] = true

			if c, ok := known[key]; ok {
				next = append(next, c)
				continue
			}

			typ := strings.TrimSpace(typeSplit.Split(rest, 2)[0])
			c := column{
				DefaultVal: json.RawMessage("null"),
				TableName:  table,
				ColumnName: name,
				Type:       strings.ReplaceAll(typ, "[]", ""),
				Nullable:   !strings.Contains(rest, "NOT NULL"),
				Array:      strings.Contains(typ, "[]"),
			}

			if dm := defaultPattern.FindStringSubmatch(rest); dm != nil {
				def := dm[1]
				c.DefaultSQL = &def
			}

			next = append(next, c)
			added = append(added, key)
		}
	}

	for _, c := range existing {
		key := c.TableName + "." + c.ColumnName

		if !schemaTables[c.TableName] {
			next = append(next, c)
			continue
		}

		if !seen[key] && !slices.Contains(skipTables, c.TableName) {
			removed = append(removed, key)
		}
	}

	return next, added, removed
}
