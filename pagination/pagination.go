// Copyright (C) 2026 NodeByte LTD

package pagination

import (
	"net/http"
	"strconv"
)

func Parse(r *http.Request) (uint64, error) {
	page := r.URL.Query().Get("page")

	if page == "" {
		return 1, nil
	}

	return strconv.ParseUint(page, 10, 32)
}
