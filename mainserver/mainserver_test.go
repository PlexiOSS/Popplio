// Copyright (C) 2026 NodeByte LTD

package mainserver

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/disgoorg/disgo/rest"
)

func TestIsNotFound(t *testing.T) {
	notFound := &http.Response{StatusCode: http.StatusNotFound}
	forbidden := &http.Response{StatusCode: http.StatusForbidden}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"value 404 from disgo", rest.NewError(nil, nil, notFound, []byte(`{"code":10007,"message":"Unknown Member"}`)), true},
		{"wrapped value 404", fmt.Errorf("get member: %w", rest.NewError(nil, nil, notFound, nil)), true},
		{"pointer 404", &rest.Error{Response: notFound}, true},
		{"value 403", rest.NewError(nil, nil, forbidden, nil), false},
		{"no response", rest.Error{}, false},
		{"unrelated error", errors.New("boom"), false},
	}

	for _, c := range cases {
		if got := isNotFound(c.err); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
