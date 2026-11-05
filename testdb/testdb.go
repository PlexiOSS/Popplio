package testdb

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"os"
	"strconv"
	"testing"

	"popplio/state"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func URL() string {
	if url := os.Getenv("POPPLIO_TEST_DATABASE_URL"); url != "" {
		return url
	}

	return os.Getenv("ARCADIA_TEST_DATABASE_URL")
}

func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if state.Pool != nil {
		return state.Pool
	}

	url := URL()

	if url == "" {
		t.Skip("POPPLIO_TEST_DATABASE_URL is not set; skipping database tests")
	}

	pool, err := pgxpool.New(context.Background(), url)

	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	state.Pool = pool

	if state.Logger == nil {
		state.Logger = zap.NewNop()
	}

	return pool
}

func ID(t *testing.T) string {
	t.Helper()

	var b [8]byte

	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}

	return strconv.FormatUint(binary.BigEndian.Uint64(b[:])%9e17+1e17, 10)
}

func Exec(t *testing.T, sql string, args ...any) {
	t.Helper()

	if _, err := state.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func Cleanup(t *testing.T, sql string, args ...any) {
	t.Helper()

	t.Cleanup(func() {
		_, _ = state.Pool.Exec(context.Background(), sql, args...)
	})
}
