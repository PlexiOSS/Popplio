package dbmigrate

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const Dir = "db/migrations"

func Apply(dsn string) error {
	db, err := goose.OpenDBWithDriver("postgres", dsn)

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	if err := goose.Up(db, Dir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func Status(dsn string) error {
	db, err := goose.OpenDBWithDriver("postgres", dsn)

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	if err := goose.Status(db, Dir); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	return nil
}
