// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"errors"
	"net/url"
	"strings"

	"popplio/state"
)

func safeJoinPopplio(rawPath string) (string, error) {
	base, err := url.Parse(state.Config.Sites.API)

	if err != nil {
		return "", err
	}

	ref, err := url.Parse(rawPath)

	if err != nil {
		return "", err
	}

	if ref.Scheme != "" || ref.Host != "" || ref.Opaque != "" {
		return "", errors.New("path must not contain a scheme or host")
	}

	if !strings.HasPrefix(ref.Path, "/") {
		return "", errors.New("path must start with /")
	}

	resolved := base.ResolveReference(ref)

	if resolved.Scheme != base.Scheme || resolved.Host != base.Host {
		return "", errors.New("path escapes the popplio base url")
	}

	return resolved.String(), nil
}
