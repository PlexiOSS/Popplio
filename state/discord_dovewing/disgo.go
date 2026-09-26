// Copyright (C) 2026 NodeByte LTD

package discord_dovewing

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"popplio/japi"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

func disgoFlagsToArray(u *discord.User) []string {
	var arr = []string{}

	if u.Bot {
		if u.PublicFlags.Has(discord.UserFlagBotHTTPInteractions) {
			arr = append(arr, "BOT_HTTP_INTERACTIONS")
		}

		if u.PublicFlags.Has(discord.UserFlagVerifiedBot) {
			arr = append(arr, "VERIFIED_BOT")
		}
	}

	return arr
}

func disgoPlatformStatus(status discord.OnlineStatus) dovetypes.PlatformStatus {
	switch status {
	case discord.OnlineStatusOnline:
		return dovetypes.PlatformStatusOnline
	case discord.OnlineStatusIdle:
		return dovetypes.PlatformStatusIdle
	case discord.OnlineStatusDND:
		return dovetypes.PlatformStatusDoNotDisturb
	default:
		return dovetypes.PlatformStatusOffline
	}
}

type DisgoState struct {
	config      *DisgoStateConfig
	initialized bool
}

type DisgoStateConfig struct {
	Client         bot.Client
	PreferredGuild *snowflake.ID
	BaseState      *dovewing.BaseState
}

func (c DisgoStateConfig) New() (*DisgoState, error) {
	if c.Client == nil {
		return nil, errors.New("discord not enabled")
	}

	if c.BaseState == nil {
		return nil, errors.New("base state not provided")
	}

	return &DisgoState{
		config: &c,
	}, nil
}

func (d *DisgoState) PlatformName() string {
	return "discord"
}

func (d *DisgoState) Init() error {
	d.initialized = true
	return nil
}

func (d *DisgoState) Initted() bool {
	return d.initialized
}

func (d *DisgoState) GetState() *dovewing.BaseState {
	return d.config.BaseState
}

func (d *DisgoState) ValidateId(id string) (string, error) {
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return "", err
	}

	if len(id) <= 16 || len(id) > 20 {
		return "", errors.New("invalid snowflake")
	}

	return id, nil
}

func (d *DisgoState) PlatformSpecificCache(ctx context.Context, idStr string) (*dovetypes.PlatformUser, error) {
	id, err := snowflake.Parse(idStr)

	if err != nil {
		return nil, err
	}

	if d.config.PreferredGuild != nil {
		member, ok := d.config.Client.Caches().Member(*d.config.PreferredGuild, id)

		if ok {
			p, pOk := d.config.Client.Caches().Presence(*d.config.PreferredGuild, id)

			var status = discord.OnlineStatusOffline
			if pOk {
				status = p.Status
			}

			return &dovetypes.PlatformUser{
				ID:          idStr,
				Username:    member.User.Username,
				Avatar:      member.User.EffectiveAvatarURL(),
				DisplayName: member.EffectiveName(),
				Bot:         member.User.Bot,
				Flags:       disgoFlagsToArray(&member.User),
				ExtraData: map[string]any{
					"cache":           "platform",
					"nickname":        member.Nick,
					"mutual_guild":    d.config.PreferredGuild,
					"preferred_guild": true,
					"public_flags":    member.User.PublicFlags,
				},
				Status: disgoPlatformStatus(status),
			}, nil
		}
	}

	var puser *dovetypes.PlatformUser
	d.config.Client.Caches().GuildCache().ForEach(func(guild discord.Guild) {
		if puser != nil || err != nil {
			return
		}

		member, ok := d.config.Client.Caches().Member(guild.ID, id)

		if ok {
			p, pOk := d.config.Client.Caches().Presence(guild.ID, id)

			var status = discord.OnlineStatusOffline
			if pOk {
				status = p.Status
			}

			puser = &dovetypes.PlatformUser{
				ID:          idStr,
				Username:    member.User.Username,
				Avatar:      member.User.EffectiveAvatarURL(),
				DisplayName: member.EffectiveName(),
				Bot:         member.User.Bot,
				Flags:       disgoFlagsToArray(&member.User),
				ExtraData: map[string]any{
					"cache":           "platform",
					"nickname":        member.Nick,
					"mutual_guild":    guild.ID.String(),
					"preferred_guild": false,
					"public_flags":    member.User.PublicFlags,
				},
				Status: disgoPlatformStatus(status),
			}
			err = nil
		}
	})

	return puser, err
}

const (
	discordLookupTimeout = 5 * time.Second
	discordSlowBackoff   = 30 * time.Second
)

var (
	errDiscordSlow       = errors.New("discord user lookup timed out")
	errDiscordBackingOff = errors.New("discord user lookups backing off after a timeout")
	discordBackoffUntil  atomic.Int64
)

type discordLookup struct {
	user *discord.User
	err  error
}

func (d *DisgoState) lookupDiscordUser(ctx context.Context, id snowflake.ID) (*discord.User, error) {
	if time.Now().UnixNano() < discordBackoffUntil.Load() {
		return nil, errDiscordBackingOff
	}

	done := make(chan discordLookup, 1)

	go func() {
		user, err := d.config.Client.Rest().GetUser(id)
		done <- discordLookup{user: user, err: err}
	}()

	timer := time.NewTimer(discordLookupTimeout)
	defer timer.Stop()

	select {
	case r := <-done:
		return r.user, r.err
	case <-timer.C:
		discordBackoffUntil.Store(time.Now().Add(discordSlowBackoff).UnixNano())
		return nil, errDiscordSlow
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (d *DisgoState) GetUser(ctx context.Context, idStr string) (*dovetypes.PlatformUser, error) {
	id, err := snowflake.Parse(idStr)

	if err != nil {
		return nil, err
	}

	user, err := d.lookupDiscordUser(ctx, id)

	if err != nil {
		if isDefinitiveMiss(err) {
			return nil, err
		}

		fallback, japiErr := japi.GetUser(ctx, idStr)

		if japiErr != nil {
			return nil, errors.Join(err, japiErr)
		}

		d.config.BaseState.Logger.Info("Resolved user via japi.rest fallback", zap.String("id", idStr), zap.NamedError("discord_error", err))

		return &dovetypes.PlatformUser{
			ID:          idStr,
			Username:    fallback.Username,
			Avatar:      fallback.EffectiveAvatarURL(),
			DisplayName: fallback.EffectiveName(),
			Bot:         fallback.Bot,
			Status:      dovetypes.PlatformStatusOffline,
			Flags:       japiFlagsToArray(fallback),
			ExtraData: map[string]any{
				"cache": "japi",
			},
		}, nil
	}

	return &dovetypes.PlatformUser{
		ID:          idStr,
		Username:    user.Username,
		Avatar:      user.EffectiveAvatarURL(),
		DisplayName: user.EffectiveName(),
		Bot:         user.Bot,
		Status:      dovetypes.PlatformStatusOffline,
		Flags:       disgoFlagsToArray(user),
	}, nil
}

func isDefinitiveMiss(err error) bool {
	var restErr rest.Error

	if !errors.As(err, &restErr) || restErr.Response == nil {
		return false
	}

	switch restErr.Response.StatusCode {
	case http.StatusNotFound, http.StatusBadRequest:
		return true
	default:
		return false
	}
}

func japiFlagsToArray(u *japi.User) []string {
	var arr = []string{}

	if !u.Bot {
		return arr
	}

	for _, flag := range u.PublicFlagsArray {
		switch flag {
		case "BOT_HTTP_INTERACTIONS", "VERIFIED_BOT":
			arr = append(arr, flag)
		}
	}

	return arr
}
