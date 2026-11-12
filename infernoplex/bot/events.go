// Copyright (C) 2026 NodeByte LTD

package bot

import (
	"context"

	"popplio/db"
	"popplio/state"
	"popplio/trackingbot"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

func onReady(_ context.Context, e *events.Ready) {
	state.Logger.Info("Infernoplex is ready!", zap.String("username", e.User.Username))

	loadOwners()
}

func onGuildMemberCount(ctx context.Context, guild discord.Guild) {
	if guild.MemberCount <= 0 {
		return
	}

	if err := db.New(state.Pool).SyncServerTotalMembers(ctx, db.SyncServerTotalMembersParams{
		ServerID:     guild.ID.String(),
		TotalMembers: int32(guild.MemberCount),
	}); err != nil {
		state.Logger.Warn("Failed to record guild member count", zap.Error(err), zap.String("server_id", guild.ID.String()))
	}
}

func onTrackingBotJoin(ctx context.Context, guildID snowflake.ID) {
	if err := trackingbot.Present(ctx, guildID.String()); err != nil {
		state.Logger.Warn("trackingbot: failed to clear missing state on join", zap.Error(err), zap.Stringer("guild_id", guildID))
	}
}

func onTrackingBotLeave(ctx context.Context, guildID snowflake.ID) {
	if err := trackingbot.Missing(ctx, guildID.String()); err != nil {
		state.Logger.Warn("trackingbot: failed to mark server missing on leave", zap.Error(err), zap.Stringer("guild_id", guildID))
	}
}
