// Copyright (C) 2026 NodeByte LTD

package bot

import (
	"context"

	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

func onReady(_ context.Context, e *events.Ready) {
	state.Logger.Info("Infernoplex is ready!", zap.String("username", e.User.Username))

	loadOwners()

	if err := SyncCommands(); err != nil {
		state.Logger.Error("Failed to sync Infernoplex application commands", zap.Error(err))
	}
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
