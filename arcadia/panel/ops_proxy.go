package panel

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"

	"popplio/arcadia/types"
	"popplio/state"
)

type clientIPKey struct{}

func withClientIP(ctx context.Context, r *http.Request) context.Context {
	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		host = r.RemoteAddr
	}

	return context.WithValue(ctx, clientIPKey{}, host)
}

func clientIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func (s *Server) popplioStaff(ctx context.Context, q *types.QPopplioStaff) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !validHTTPMethod(q.Method) {
		return writeText(http.StatusBadRequest, "Invalid method"), nil
	}

	if !strings.HasPrefix(q.Path, "/") {
		return writeText(http.StatusBadRequest, "Path must start with /"), nil
	}

	target, err := safeJoinPopplio(q.Path)

	if err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	var popplioToken string

	err = state.Pool.QueryRow(ctx, "SELECT popplio_token FROM staffpanel__authchain WHERE token = $1", q.LoginToken).Scan(&popplioToken)

	if err != nil {
		return response{}, newError(err)
	}

	req, err := http.NewRequestWithContext(ctx, q.Method, target, strings.NewReader(q.Body))

	if err != nil {
		return response{}, newError(err)
	}

	req.Header.Set("User-Agent", "arcadia")

	if ip := clientIPFromContext(ctx); ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
	}
	req.Header.Set("X-Staff-Auth-Token", popplioToken)
	req.Header.Set("X-User-ID", authData.UserID)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return response{}, newError(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return response{}, newError(err)
	}

	return writeText(resp.StatusCode, string(body)), nil
}

func validHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect,
		http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
