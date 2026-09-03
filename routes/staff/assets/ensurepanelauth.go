// Package assets holds the staff-panel authorization checks.
package assets

import (
	"context"
	"errors"
	"net/http"

	"popplio/db"
	"popplio/state"
)

func EnsurePanelAuth(ctx context.Context, r *http.Request) (uid string, err error) {
	ssToken := r.Header.Get("X-Staff-Auth-Token")
	userId := r.Header.Get("X-User-ID")

	if ssToken == "" {
		return "", errors.New("missing staff auth token normally sent by Arcadia")
	}

	if userId == "" {
		return "", errors.New("missing user id header")
	}

	count, err := db.New(state.Pool).CountAuthChainToken(ctx, db.CountAuthChainTokenParams{
		PopplioToken: ssToken,
		UserID:       userId,
	})

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", errors.New("identityExpired")
	}

	return userId, nil
}
