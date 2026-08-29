// Command migrate applies Popplio's SQL schema migrations.
//
// This is the replacement for hand-running exp/*.sql files against prod via
// psql -- new schema changes belong in ../../db/migrations as versioned,
// goose-tracked .sql files instead. It deliberately loads only enough of
// Popplio's config to get a Postgres DSN (unlike state.Setup(), which also
// dials Discord, Redis, Stripe, and PayPal -- none of which a migration run
// needs, and connecting to them here would just be extra ways for this to
// fail or hang).
//
// Usage:
//
//	go run ./cmd/migrate status
//	go run ./cmd/migrate up
//	go run ./cmd/migrate create add_some_column
//	go run ./cmd/migrate validate
package main

import (
	"fmt"
	"os"

	"popplio/config"
	"popplio/db/dbmigrate"

	"github.com/pressly/goose/v3"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "create" {
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: migrate create <name>")
			os.Exit(1)
		}

		if err := goose.Create(nil, dbmigrate.Dir, os.Args[2], "sql"); err != nil {
			fatal(err)
		}

		return
	}

	dsn, err := loadPostgresURL()

	if err != nil {
		fatal(err)
	}

	switch command {
	case "up":
		err = dbmigrate.Apply(dsn)
	case "status":
		err = dbmigrate.Status(dsn)
	case "validate":
		err = runValidate(dsn)
	default:
		usage()
		os.Exit(1)
	}

	if err != nil {
		fatal(err)
	}
}

func runValidate(dsn string) error {
	findings, err := dbmigrate.Validate(dsn, tableSchemas)

	if err != nil {
		return err
	}

	if len(findings) == 0 {
		fmt.Printf("OK: %d table(s) checked, no drift found\n", len(tableSchemas))
		return nil
	}

	fmt.Printf("Found %d issue(s) across %d table(s) checked:\n", len(findings), len(tableSchemas))

	for _, f := range findings {
		fmt.Println("  " + f.String())
	}

	os.Exit(1)
	return nil
}

// loadPostgresURL prefers DATABASE_URL when set -- lets this point at a
// scratch/test database (e.g. for a dry run of every migration) without
// ever touching the real config.yaml, which is what the actual running
// server reads.
func loadPostgresURL() (string, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	raw, err := os.ReadFile("config.yaml")

	if err != nil {
		return "", fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var cfg config.Config

	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	if cfg.Meta.PostgresURL == "" {
		return "", fmt.Errorf("config.yaml has no meta.postgres_url set")
	}

	return cfg.Meta.PostgresURL, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate <up|status|validate|create> [args]")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "migrate: "+err.Error())
	os.Exit(1)
}
