package validators

import "net/url"

// IsNonProdFrontend reports whether the given frontend URL (an OAuth
// redirect_uri, or a request's Origin header) points somewhere other than
// the configured production frontend (Config.Sites.Frontend).
//
// This replaces the old build-time CurrentEnv-based staging/beta detection:
// there's only one deployed Popplio instance now, but multiple frontends
// (a staging/beta site, local dev) can still point at it, and some checks
// — like restricting sign-in to Bug Hunters — need to key off which one is
// actually calling, not which binary is running. An empty or unparseable
// value is treated as non-prod, since the caller couldn't prove it was the
// real frontend.
func IsNonProdFrontend(candidate, prodFrontend string) bool {
	if candidate == "" {
		return true
	}

	prodURL, err := url.Parse(prodFrontend)
	if err != nil || prodURL.Host == "" {
		return true
	}

	candidateURL, err := url.Parse(candidate)
	if err != nil {
		return true
	}

	return candidateURL.Host != prodURL.Host
}
