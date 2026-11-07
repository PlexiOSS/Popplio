// Copyright (C) 2026 NodeByte LTD

package boosters

import (
	"errors"
	"slices"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

var ErrNotReady = errors.New("the Discord member cache for the main server is empty")

func IsBooster(m discord.Member) bool {
	if m.User.Bot {
		return false
	}

	if m.PremiumSince != nil {
		return true
	}

	for _, role := range m.RoleIDs {
		if slices.Contains(state.Config.Roles.PremiumRoles, role) {
			return true
		}
	}

	return false
}

func Check(userID snowflake.ID) (member, boosting bool) {
	m, ok := state.Discord.Caches().Member(state.Config.Servers.Main, userID)

	if !ok {
		return false, false
	}

	return true, IsBooster(m)
}

func List() ([]string, error) {
	ids := []string{}
	seen := 0

	state.Discord.Caches().MembersForEach(state.Config.Servers.Main, func(m discord.Member) {
		seen++

		if IsBooster(m) {
			ids = append(ids, m.User.ID.String())
		}
	})

	if seen == 0 {
		return nil, ErrNotReady
	}

	return ids, nil
}
