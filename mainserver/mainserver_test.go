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
		{"404 from disgo", &rest.Error{Response: notFound, Code: 10007, Message: "Unknown Member"}, true},
		{"wrapped 404", fmt.Errorf("get member: %w", &rest.Error{Response: notFound}), true},
		{"403", &rest.Error{Response: forbidden}, false},
		{"no response", &rest.Error{}, false},
		{"unrelated error", errors.New("boom"), false},
	}

	for _, c := range cases {
		if got := isNotFound(c.err); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
