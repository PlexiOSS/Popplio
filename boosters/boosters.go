// Copyright (C) 2026 NodeByte LTD

package boosters

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

var ErrNotReady = errors.New("the Discord member cache for the main server is empty")

var rechunking atomic.Bool

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

func Lookup(userID snowflake.ID) (member, boosting bool, err error) {
	if m, ok := state.Discord.Caches.Member(state.Config.Servers.Main, userID); ok {
		return true, IsBooster(m), nil
	}

	m, err := state.Discord.Rest.GetMember(state.Config.Servers.Main, userID)

	if err != nil {
		var restErr *rest.Error

		if errors.As(err, &restErr) && restErr.Response != nil && restErr.Response.StatusCode == http.StatusNotFound {
			return false, false, nil
		}

		return false, false, err
	}

	m.GuildID = state.Config.Servers.Main
	state.Discord.Caches.AddMember(*m)

	return true, IsBooster(*m), nil
}

func Check(userID snowflake.ID) (member, boosting bool) {
	member, boosting, err := Lookup(userID)

	if err != nil {
		state.Logger.Warn("boosters: failed to look up member", zap.Error(err), zap.String("user_id", userID.String()))
	}

	return member, boosting
}

func List() ([]string, error) {
	ids := []string{}
	seen := 0

	for m := range state.Discord.Caches.Members(state.Config.Servers.Main) {
		seen++

		if IsBooster(m) {
			ids = append(ids, m.User.ID.String())
		}
	}

	if seen == 0 {
		return nil, ErrNotReady
	}

	if guild, ok := state.Discord.Caches.Guild(state.Config.Servers.Main); ok && seen < guild.MemberCount*9/10 {
		rechunk(seen, guild.MemberCount)
	}

	return ids, nil
}

func StillBoosting(userIDs []string) []string {
	still := []string{}

	for _, id := range userIDs {
		sf, err := snowflake.Parse(id)

		if err != nil {
			continue
		}

		_, boosting, err := Lookup(sf)

		if err != nil {
			state.Logger.Warn("boosters: couldn't confirm a booster stopped, keeping them for now", zap.Error(err), zap.String("user_id", id))
			still = append(still, id)
			continue
		}

		if boosting {
			still = append(still, id)
		}
	}

	return still
}

func rechunk(cached, total int) {
	if !rechunking.CompareAndSwap(false, true) {
		return
	}

	state.Logger.Warn("boosters: member cache for the main server is incomplete, requesting members again", zap.Int("cached", cached), zap.Int("member_count", total))

	go func() {
		defer rechunking.Store(false)

		ctx, cancel := context.WithTimeout(state.Context, 2*time.Minute)
		defer cancel()

		if _, err := state.Discord.MemberChunkingManager.RequestAllMembers(ctx, state.Config.Servers.Main); err != nil {
			state.Logger.Error("boosters: re-requesting main server members failed", zap.Error(err))
		}
	}()
}
