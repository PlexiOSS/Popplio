// Copyright (C) 2026 NodeByte LTD

package main

import (
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"popplio/audit"
	"popplio/brandemoji"
	"popplio/linkedroles"
	"runtime"
	"strings"
	"syscall"
	"time"

	"popplio/api"
	"popplio/arcadia"
	"popplio/bgtasks"
	"popplio/constants"
	"popplio/db/dbmigrate"
	"popplio/infernoplex"
	"popplio/notifications/votereminders"
	"popplio/routes/all"
	"popplio/state"
	poplhooks "popplio/webhooks"

	"github.com/cloudflare/tableflip"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/PlexiOSS/Keel/urlutil"

	"github.com/PlexiOSS/Keel/zapchi"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "embed"
)

//go:embed data/docs.html
var docsHTML string

var openapi []byte

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024)

		if r.Header.Get("User-Auth") != "" {
			if strings.HasPrefix(r.Header.Get("User-Auth"), "User ") {
				r.Header.Set("Authorization", r.Header.Get("User-Auth"))
			} else {
				r.Header.Set("Authorization", "User "+r.Header.Get("User-Auth"))
			}
		} else if r.Header.Get("Bot-Auth") != "" {
			if strings.HasPrefix(r.Header.Get("Bot-Auth"), "Bot ") {
				r.Header.Set("Authorization", r.Header.Get("Bot-Auth"))
			}
			r.Header.Set("Authorization", "Bot "+r.Header.Get("Bot-Auth"))
		}

		origin := r.Header.Get("Origin")

		if origin == "" {
			origin = state.Config.Sites.Frontend
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		if !urlutil.DifferentHost(origin, state.Config.Sites.Frontend) {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Headers", "X-Client, Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "X-Session-Invalid, Retry-After")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		if r.Method == "OPTIONS" {
			w.Write([]byte{})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		next.ServeHTTP(w, r)
	})
}

func main() {
	state.Setup()
	brandemoji.LoadAsync(brandemoji.Popplio, state.Discord, "popplio")
	linkedroles.RegisterMetadataAsync()

	if !state.Config.Meta.SkipAutoMigrate {
		applied, err := dbmigrate.ApplyEmbedded(state.Context, state.Config.Meta.PostgresURL)

		for _, path := range applied {
			state.Logger.Info("Applied database migration", zap.String("migration", path))
		}

		if err != nil {
			state.Logger.Fatal("Database migrations failed, refusing to start", zap.Error(err))
		}
	}

	var err error

	all.SetupDocs(state.Config.Sites.API)
	poplhooks.Setup()

	api.Setup()

	r := chi.NewRouter()

	r.Use(
		middleware.Recoverer,
		middleware.RealIP,
		middleware.CleanPath,
		corsMiddleware,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
		audit.Middleware,
	)

	routers := all.Routers()

	for _, router := range routers {
		name, desc := router.Tag()
		if name != "" {
			docs.AddTag(name, desc)
			uapi.State.SetCurrentTag(name)
		} else {
			panic("Router tag name cannot be empty")
		}

		router.Routes(r)
	}

	r.Get("/openapi", func(w http.ResponseWriter, r *http.Request) {
		w.Write(openapi)
	})

	docsTempl := template.Must(template.New("docs").Parse(docsHTML))

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/popplio", http.StatusFound)
	})

	r.Get("/docs/{srv}", func(w http.ResponseWriter, r *http.Request) {
		docMap := map[string]string{
			"popplio":     "/openapi",
			"arcadia":     "https://prod--panel-api.omniplex.gg/openapi",
			"infernoplex": "https://infernoplex.omniplex.gg/openapi",
		}

		srv := chi.URLParam(r, "srv")

		if srv == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid service name"))
			return
		}

		v, ok := docMap[srv]

		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid service"))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		docsTempl.Execute(w, map[string]string{
			"url": v,
		})
	})

	openapi, err = jsonimpl.Marshal(docs.GetSchema())

	if err != nil {
		panic(err)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(constants.EndpointNotFound))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(constants.MethodNotAllowed))
	})

	go votereminders.VrLoop()

	bgtasks.Start(state.Context)

	arc := arcadia.Start(state.Context)
	defer arc.Stop(30 * time.Second)

	inf := infernoplex.Start(state.Context)
	defer inf.Stop(30 * time.Second)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		upg, _ := tableflip.New(tableflip.Options{})
		defer upg.Stop()

		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGHUP)
			for range sig {
				state.Logger.Info("Received SIGHUP, upgrading server")
				upg.Upgrade()
			}
		}()

		ln, err := upg.Listen("tcp", state.Config.Meta.Port)

		if err != nil {
			state.Logger.Fatal("Error binding to socket", zap.Error(err))
		}

		defer ln.Close()

		server := http.Server{
			ReadTimeout: 30 * time.Second,
			Handler:     r,
		}

		go func() {
			err := server.Serve(ln)
			if err != http.ErrServerClosed {
				state.Logger.Error("Server failed due to unexpected error", zap.Error(err))
			}
		}()

		if err := upg.Ready(); err != nil {
			state.Logger.Fatal("Error calling upg.Ready", zap.Error(err))
		}

		<-upg.Exit()
	} else {
		state.Logger.Warn("Tableflip not supported on this platform, this is not a production-capable server.")
		err = http.ListenAndServe(state.Config.Meta.Port, r)

		if err != nil {
			state.Logger.Fatal("Error binding to socket", zap.Error(err))
		}
	}
}
