// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"errors"
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
		{"value 403", rest.NewError(nil, nil, &http.Response{StatusCode: http.StatusForbidden}, nil), true},
		{"value 404", rest.NewError(nil, nil, &http.Response{StatusCode: http.StatusNotFound}, nil), true},
		{"pointer 404", &rest.Error{Response: &http.Response{StatusCode: http.StatusNotFound}}, true},
		{"value 500", rest.NewError(nil, nil, &http.Response{StatusCode: http.StatusInternalServerError}, nil), false},
		{"unrelated", errors.New("boom"), false},
	}

	for _, c := range cases {
		if got := isMissingAccess(c.err); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
