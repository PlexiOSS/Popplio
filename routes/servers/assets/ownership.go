// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"errors"
	"net/http"
	"sync"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

var (
	ErrTrackingBotMissing = errors.New("invite our tracking bot to this server first, so we can confirm you manage it")
	ErrNotServerManager   = errors.New("you need to own this server, or have the Manage Server or Administrator permission in it, to list it")
)

var (
	trackingRestOnce sync.Once
	trackingRest     rest.Rest
)

func trackingBotRest() rest.Rest {
	trackingRestOnce.Do(func() {
		trackingRest = rest.New(rest.NewClient(state.Config.Infernoplex.Token, state.ProxyRestOpts(state.Config.Infernoplex.Token)...))
	})

	return trackingRest
}

func isMissingAccess(err error) bool {
	var restErr *rest.Error

	if errors.As(err, &restErr) && restErr.Response != nil {
		switch restErr.Response.StatusCode {
		case http.StatusForbidden, http.StatusNotFound:
			return true
		}
	}

	return false
}

func VerifyServerManager(guildID, userID string) error {
	gid, err := snowflake.Parse(guildID)

	if err != nil {
		return err
	}

	uid, err := snowflake.Parse(userID)

	if err != nil {
		return err
	}

	client := trackingBotRest()

	guild, err := client.GetGuild(gid, false)

	if err != nil {
		if isMissingAccess(err) {
			return ErrTrackingBotMissing
		}

		return err
	}

	if guild.OwnerID == uid {
		return nil
	}

	member, err := client.GetMember(gid, uid)

	if err != nil {
		if isMissingAccess(err) {
			return ErrNotServerManager
		}

		return err
	}

	held := make(map[snowflake.ID]bool, len(member.RoleIDs)+1)
	held[gid] = true

	for _, id := range member.RoleIDs {
		held[id] = true
	}

	var perms discord.Permissions

	for _, role := range guild.Roles {
		if held[role.ID] {
			perms |= role.Permissions
		}
	}

	if perms.Has(discord.PermissionAdministrator) || perms.Has(discord.PermissionManageGuild) {
		return nil
	}

	return ErrNotServerManager
}
