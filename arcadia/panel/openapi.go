// Copyright (C) 2026 NodeByte LTD

package panel

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var openapiDoc []byte

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeText(http.StatusMethodNotAllowed, "Method Not Allowed").write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(openapiDoc)
}
