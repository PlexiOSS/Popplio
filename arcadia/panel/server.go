// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"popplio/arcadia/types"
	"popplio/state"

	"github.com/go-chi/chi/v5/middleware"

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
				middleware.RealIP(
					maxBodyMiddleware(mux),
				),
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

	body, err := io.ReadAll(r.Body)

	if err != nil {
		writeText(http.StatusBadRequest, "Failed to read the request body").write(w)
		return
	}

	body, err = injectCookieToken(r, body)

	if err != nil {
		writeText(http.StatusForbidden, err.Error()).write(w)
		return
	}

	var req types.PanelQuery

	if err := json.Unmarshal(body, &req); err != nil {
		writeText(http.StatusBadRequest, "Failed to parse the request body as JSON: "+err.Error()).write(w)
		return
	}

	ctx := withClientIP(r.Context(), r)
	resp, err := s.dispatch(ctx, &req)

	if err != nil {
		var perr Error

		status := http.StatusInternalServerError
		message := err.Error()

		if errors.As(err, &perr) {
			status, message = perr.Status, perr.Message
		}

		recordPanel(ctx, body, status)
		writeText(status, message).write(w)
		return
	}

	status := resp.status

	if status == 0 {
		status = http.StatusOK
	}

	recordPanel(ctx, body, status)
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
		if origin := r.Header.Get("Origin"); allowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		}

		w.Header().Add("Vary", "Origin")

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
