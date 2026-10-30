package sessioncookie

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"popplio/state"
)

const Name = "popplio_session"

const lifetime = 30 * 24 * time.Hour

func Domain(frontend, api string) string {
	f, err := url.Parse(frontend)
	if err != nil {
		return ""
	}
	a, err := url.Parse(api)
	if err != nil {
		return ""
	}

	fh, ah := f.Hostname(), a.Hostname()
	if net.ParseIP(fh) != nil || net.ParseIP(ah) != nil {
		return ""
	}

	fl, al := strings.Split(fh, "."), strings.Split(ah, ".")
	shared := []string{}
	for i := 1; i <= len(fl) && i <= len(al); i++ {
		if fl[len(fl)-i] != al[len(al)-i] {
			break
		}
		shared = append([]string{fl[len(fl)-i]}, shared...)
	}

	if len(shared) < 2 {
		return ""
	}

	return strings.Join(shared, ".")
}

func build(value string, maxAge int) string {
	c := &http.Cookie{
		Name:     Name,
		Value:    value,
		Path:     "/",
		Domain:   Domain(state.Config.Sites.Frontend, state.Config.Sites.API),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   strings.HasPrefix(state.Config.Sites.Frontend, "https://"),
		SameSite: http.SameSiteLaxMode,
	}

	return c.String()
}

func Set(token string) string {
	return build(token, int(lifetime.Seconds()))
}

func Clear() string {
	return build("", -1)
}

func Token(r *http.Request) string {
	c, err := r.Cookie(Name)
	if err != nil {
		return ""
	}

	return c.Value
}
