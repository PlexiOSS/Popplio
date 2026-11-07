// Copyright (C) 2026 NodeByte LTD

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"popplio/api"
	"popplio/config"
	"popplio/routes/all"
	"popplio/state"

	"github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	out := "openapi.json"

	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	apiURL := os.Getenv("OPENAPI_SERVER_URL")

	if apiURL == "" {
		apiURL = "https://api.omniplex.gg"
	}

	state.Logger = zap.NewNop()
	state.Config = &config.Config{}
	state.Config.Sites.API = apiURL
	state.Config.Sites.Frontend = "https://omniplex.gg"
	state.Context = context.Background()

	all.SetupDocs(apiURL)
	api.Setup()

	r := chi.NewRouter()

	for _, router := range all.Routers() {
		name, desc := router.Tag()
		doclib.AddTag(name, desc)
		uapi.State.SetCurrentTag(name)
		router.Routes(r)
	}

	data, err := json.MarshalIndent(doclib.GetSchema(), "", "  ")

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("wrote", out)
}
