// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/disgoorg/disgo/rest"
)

func TestIsMissingAccess(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"403", &rest.Error{Response: &http.Response{StatusCode: http.StatusForbidden}}, true},
		{"404", &rest.Error{Response: &http.Response{StatusCode: http.StatusNotFound}}, true},
		{"wrapped 404", fmt.Errorf("get: %w", &rest.Error{Response: &http.Response{StatusCode: http.StatusNotFound}}), true},
		{"500", &rest.Error{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, false},
		{"unrelated", errors.New("boom"), false},
	}

	for _, c := range cases {
		if got := isMissingAccess(c.err); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
