package dbmigrate

import (
	"context"
	"fmt"

	"popplio/db/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/lock"
)

const Dir = "db/migrations"

func Apply(dsn string) error {
	_, err := ApplyEmbedded(context.Background(), dsn)
	return err
}

func Status(dsn string) error {
	db, err := goose.OpenDBWithDriver("postgres", dsn)

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	defer goose.SetBaseFS(nil)

	if err := goose.Status(db, "."); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	return nil
}

func ApplyEmbedded(ctx context.Context, dsn string) ([]string, error) {
	db, err := goose.OpenDBWithDriver("postgres", dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	locker, err := lock.NewPostgresSessionLocker()

	if err != nil {
		return nil, fmt.Errorf("failed to create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(database.DialectPostgres, db, migrations.FS,
		goose.WithSessionLocker(locker),
		goose.WithAllowOutofOrder(true),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	results, err := provider.Up(ctx)

	applied := make([]string, 0, len(results))

	for _, r := range results {
		if r.Source != nil && r.Error == nil {
			applied = append(applied, r.Source.Path)
		}
	}

	if err != nil {
		return applied, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return applied, nil
}
