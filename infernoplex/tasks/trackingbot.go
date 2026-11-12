// Copyright (C) 2026 NodeByte LTD

package tasks

import (
	"context"
	"errors"
	"net/http"
	"time"

	"popplio/infernoplex/dclient"
	"popplio/trackingbot"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

const trackingBotCheckGap = time.Second

func TrackingBotCheck(ctx context.Context) error {
	first := true

	return trackingbot.Sweep(ctx, func(ctx context.Context, serverID string) (bool, error) {
		if !first {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(trackingBotCheckGap):
			}
		}
		first = false

		guildID, err := snowflake.Parse(serverID)

		if err != nil {
			return false, err
		}

		_, err = dclient.Get().Rest().GetGuild(guildID, false)

		if err == nil {
			return true, nil
		}

		if noAccess(err) {
			return false, nil
		}

		return false, err
	})
}

func noAccess(err error) bool {
	var value rest.Error
	var ptr *rest.Error

	var resp *http.Response

	switch {
	case errors.As(err, &value):
		resp = value.Response
	case errors.As(err, &ptr) && ptr != nil:
		resp = ptr.Response
	}

	return resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
}
