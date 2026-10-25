// Copyright (C) 2026 NodeByte LTD

package pagination

import (
	"errors"
	"net/http"
	"strconv"
)

const MaxPage = 100000

var ErrPageOutOfRange = errors.New("page must be between 1 and 100000")

func Parse(r *http.Request) (uint64, error) {
	page := r.URL.Query().Get("page")

	if page == "" {
		return 1, nil
	}

	pageNum, err := strconv.ParseUint(page, 10, 32)

	if err != nil {
		return 0, err
	}

	if pageNum < 1 || pageNum > MaxPage {
		return 0, ErrPageOutOfRange
	}

	return pageNum, nil
}
