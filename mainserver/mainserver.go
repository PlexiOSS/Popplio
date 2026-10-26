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

	if _, ok := state.Discord.Caches().Member(state.Config.Servers.Main, id); ok {
		return true, nil
	}

	_, err = state.Discord.Rest().GetMember(state.Config.Servers.Main, id)

	if err == nil {
		return true, nil
	}

	var restErr *rest.Error

	if errors.As(err, &restErr) && restErr.Response != nil && restErr.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, err
}

func CachedMember(userID string) bool {
	id, err := snowflake.Parse(userID)

	if err != nil {
		return false
	}

	_, ok := state.Discord.Caches().Member(state.Config.Servers.Main, id)

	return ok
}
