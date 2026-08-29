// Copyright (C) 2026 NodeByte LTD

package panel

import "net/http"

func requireExists(exists bool) *response {
	if !exists {
		resp := writeText(http.StatusBadRequest, "Entry with same id does not already exist")
		return &resp
	}

	return nil
}
