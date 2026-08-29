// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"popplio/arcadia/types"
	"popplio/state"

	"go.uber.org/zap"
)

const maxBodySize = 1_048_576_000

type Server struct {
	http *http.Server
}

func New() *Server {
	s := &Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi", s.handleOpenAPI)
	mux.HandleFunc("/", s.handleRoot)

	handler := recoverMiddleware(
		loggingMiddleware(
			corsMiddleware(
				maxBodyMiddleware(mux),
			),
		),
	)

	s.http = &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", state.Config.Arcadia.ServerPort),
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Minute,
		WriteTimeout:      30 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	if err := ensureAuthchainTable(ctx); err != nil {
		return fmt.Errorf("failed to create staffpanel__authchain table: %w", err)
	}

	state.Logger.Info("Starting panel server", zap.String("addr", s.http.Addr))

	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func ensureAuthchainTable(ctx context.Context) error {
	_, err := state.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS staffpanel__authchain (
            itag UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
            user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
            token TEXT NOT NULL,
            popplio_token TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            state TEXT NOT NULL DEFAULT 'pending'
        )`)

	return err
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeText(http.StatusNotFound, "Not Found").write(w)
		return
	}

	if r.Method != http.MethodPost {
		writeText(http.StatusMethodNotAllowed, "Method Not Allowed").write(w)
		return
	}

	var req types.PanelQuery

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeText(http.StatusBadRequest, "Failed to parse the request body as JSON: "+err.Error()).write(w)
		return
	}

	resp, err := s.dispatch(withClientIP(r.Context(), r), &req)

	if err != nil {
		var perr Error

		if errors.As(err, &perr) {
			writeText(perr.Status, perr.Message).write(w)
			return
		}

		writeText(http.StatusInternalServerError, err.Error()).write(w)
		return
	}

	resp.write(w)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("panel: panic serving request",
					zap.Any("panic", rec),
					zap.String("path", r.URL.Path),
					zap.ByteString("stack", debug.Stack()),
				)

				writeText(http.StatusInternalServerError, "Internal Server Error").write(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		state.Logger.Info("panel",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
			zap.Duration("took", time.Since(start)),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func maxBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		next.ServeHTTP(w, r)
	})
}

func instanceDescription() string {
	return "Arcadia Production Panel Instance"
}

func serverIDs() types.PanelServers {
	return types.PanelServers{
		Main:    strconv.FormatUint(uint64(state.Config.Servers.Main), 10),
		Staff:   strconv.FormatUint(uint64(state.Config.Servers.Staff), 10),
		Testing: strconv.FormatUint(uint64(state.Config.Servers.Testing), 10),
	}
}
