// Copyright (C) 2026 NodeByte LTD

package main

import (
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"popplio/api"
	poplapps "popplio/apps"
	"popplio/arcadia"
	"popplio/bgtasks"
	"popplio/constants"
	"popplio/infernoplex"
	"popplio/notifications/votereminders"
	"popplio/routes/alerts"
	"popplio/routes/apps"
	"popplio/routes/auth"
	"popplio/routes/badges"
	"popplio/routes/blogs"
	"popplio/routes/bots"
	"popplio/routes/changelogs"
	"popplio/routes/diagnostics"
	"popplio/routes/emojis"
	"popplio/routes/health"
	"popplio/routes/list"
	notifrouter "popplio/routes/notifications"
	"popplio/routes/packs"
	"popplio/routes/payments"
	"popplio/routes/platform"
	"popplio/routes/reminders"
	reportsrouter "popplio/routes/reports"
	"popplio/routes/reviews"
	"popplio/routes/servers"
	"popplio/routes/servertemplates"
	"popplio/routes/shop"
	"popplio/routes/staff"
	"popplio/routes/stickers"
	"popplio/routes/tasks"
	"popplio/routes/teams"
	"popplio/routes/themes"
	"popplio/routes/tickets"
	"popplio/routes/users"
	"popplio/routes/vanity"
	"popplio/routes/votes"
	"popplio/routes/webhooks"
	"popplio/state"
	"popplio/types"
	poplhooks "popplio/webhooks"

	"github.com/cloudflare/tableflip"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/jsonimpl"
	"github.com/PlexiOSS/Keel/uapi"

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
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "X-Client, Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "X-Session-Invalid, Retry-After")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")

		if r.Method == "OPTIONS" {
			w.Write([]byte{})
			return
		}

		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)
	})
}

func main() {
	state.Setup()

	var err error

	docs.DocsSetupData = &docs.SetupData{
		URL:               state.Config.Sites.API,
		ServerDescription: "Popplio",
		ErrorStruct:       types.ApiError{},
		Info: docs.Info{
			Title:          "Omniplex API",
			TermsOfService: "https://omniplex.gg/legal/terms",
			Version:        "1.8.2",
			Description:    "RESTful API and Backend Services for Omniplex",
			Contact: docs.Contact{
				Name: "Omniplex Support",
				URL:  "https://discord.gg/AxrR4RNSFT",
			},
			License: docs.License{
				Name: "AGPL-3.0",
				URL:  "https://opensource.org/licenses/AGPL-3.0",
			},
		},
	}

	docs.Setup()
	poplhooks.Setup()
	poplapps.Setup()

	docs.AddSecuritySchema("User", "User-Auth", "Requires a user token. Should be prefixed with `User ` in `Authorization` header.")
	docs.AddSecuritySchema("Bot", "Bot-Auth", "Requires a bot token. Should be prefixed with `Bot ` in `Authorization` header.")
	docs.AddSecuritySchema("server", "Server-Auth", "Requires a server API session token. Should be prefixed with `Server ` in `Authorization` header.")
	docs.AddSecuritySchema("team", "Team-Auth", "Requires a team API session token. Should be prefixed with `Team ` in `Authorization` header.")

	api.Setup()

	r := chi.NewRouter()

	r.Use(
		middleware.Recoverer,
		middleware.RealIP,
		middleware.CleanPath,
		corsMiddleware,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
	)

	routers := []uapi.APIRouter{
		alerts.Router{},
		apps.Router{},
		auth.Router{},
		badges.Router{},
		blogs.Router{},
		bots.Router{},
		changelogs.Router{},
		diagnostics.Router{},
		emojis.Router{},
		health.Router{},
		list.Router{},
		notifrouter.Router{},
		packs.Router{},
		payments.Router{},
		platform.Router{},
		reminders.Router{},
		reportsrouter.Router{},
		reviews.Router{},
		servers.Router{},
		servertemplates.Router{},
		shop.Router{},
		staff.Router{},
		stickers.Router{},
		tasks.Router{},
		teams.Router{},
		themes.Router{},
		tickets.Router{},
		users.Router{},
		vanity.Router{},
		votes.Router{},
		webhooks.Router{},
	}

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
