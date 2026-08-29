package state

import (
	"net/http"

	"github.com/disgoorg/disgo/rest"
)

type proxyAuthTransport struct {
	token string
	base  http.RoundTripper
}

func (t *proxyAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Upstream-Authorization", "Bot "+t.token)
	return t.base.RoundTrip(req)
}

func ProxyRestOpts(token string) []rest.ConfigOpt {
	return []rest.ConfigOpt{
		rest.WithURL(Config.Meta.PopplioProxy + "/v10"),
		rest.WithHTTPClient(&http.Client{
			Transport: &proxyAuthTransport{token: token, base: http.DefaultTransport},
		}),
	}
}
