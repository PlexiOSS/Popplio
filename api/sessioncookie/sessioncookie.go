package sessioncookie

import (
	"net/http"
	"strings"
	"time"

	"popplio/state"
)

const Name = "popplio_session"

const lifetime = 30 * 24 * time.Hour

func build(value string, maxAge int) string {
	c := &http.Cookie{
		Name:     Name,
		Value:    value,
		Path:     "/",
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

func SameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == "" && r.Header.Get("Sec-Fetch-Site") == "same-origin"
}
