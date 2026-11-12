// Copyright (C) 2026 NodeByte LTD

package mainserver

import (
	"errors"
	"net/http"

	"popplio/state"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

func IsMember(userID string) (bool, error) {
	id, err := snowflake.Parse(userID)

	if err != nil {
		return false, err
	}

	if _, ok := state.Discord.Caches.Member(state.Config.Servers.Main, id); ok {
		return true, nil
	}

	_, err = state.Discord.Rest.GetMember(state.Config.Servers.Main, id)

	if err == nil {
		return true, nil
	}

	if isNotFound(err) {
		return false, nil
	}

	return false, err
}

func isNotFound(err error) bool {
	var ptr *rest.Error

	if errors.As(err, &ptr) && ptr != nil {
		return ptr.Response != nil && ptr.Response.StatusCode == http.StatusNotFound
	}

	return false
}

func CachedMember(userID string) bool {
	id, err := snowflake.Parse(userID)

	if err != nil {
		return false
	}

	_, ok := state.Discord.Caches.Member(state.Config.Servers.Main, id)

	return ok
}
