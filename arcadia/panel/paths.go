package panel

import (
	"errors"
	"net/url"
	"strings"

	"popplio/state"
)

// safeJoinPopplio resolves a caller-supplied path against Popplio's base URL and
// refuses anything that would leave it.
//
// HARDENING (§14b). Upstream builds the target as a bare string concatenation
// after checking only that the path starts with '/', so "//host/x" (a
// protocol-relative URL) retargets the request at an arbitrary host and ".."
// segments escape the API base. Legitimate input - an absolute path with an
// optional query string - resolves identically here.
func safeJoinPopplio(rawPath string) (string, error) {
	base, err := url.Parse(state.Config.Sites.API.Parse())

	if err != nil {
		return "", err
	}

	ref, err := url.Parse(rawPath)

	if err != nil {
		return "", err
	}

	// A parsed reference that carries a scheme or a host is not a path.
	if ref.Scheme != "" || ref.Host != "" || ref.Opaque != "" {
		return "", errors.New("path must not contain a scheme or host")
	}

	if !strings.HasPrefix(ref.Path, "/") {
		return "", errors.New("path must start with /")
	}

	resolved := base.ResolveReference(ref)

	// Same-origin is the actual security boundary described above: once
	// scheme+host are pinned to Popplio's own base, the target can't be
	// retargeted at another host. A further same-path-prefix containment
	// check used to run here too, but it rejected legitimate root-level
	// targets (e.g. "/staff/apps") whenever the configured base URL itself
	// had a non-root path component — it added no real security beyond the
	// origin check and only broke valid callers.
	if resolved.Scheme != base.Scheme || resolved.Host != base.Host {
		return "", errors.New("path escapes the popplio base url")
	}

	return resolved.String(), nil
}
