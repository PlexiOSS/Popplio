package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
	ptypes "popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func approve(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return approveServer(ctx, m, h)
	}

	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType     string
		claimedBy   *string
		lastClaimed *time.Time
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, last_claimed FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &lastClaimed)

	if err != nil {
		return Success{}, err
	}

	if botType != "pending" {
		return Success{}, errors.New("Entity is not pending review?")
	}

	if claimedBy == nil || *claimedBy == "" || lastClaimed == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed? Do ``/claim`` to claim this bot first!", m.TargetID)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return Success{}, err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE bots SET type = 'approved', claimed_by = NULL WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       "Approved!",
			URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, m.TargetID),
			Description: fmt.Sprintf("<@!%s> has approved <@!%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Feedback", Value: m.Reason, Inline: impls.InlineTrue()},
				{Name: "Moderator", Value: "<@!" + h.UserID + ">", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Well done, young traveller!"),
			Color:  impls.ColourGreen,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Success{}, err
	}

	impls.NotifyOwners(owners.All(), ptypes.Alert{
		Type:     ptypes.AlertTypeSuccess,
		Title:    "Bot Approved!",
		Message:  "Your bot has been approved and is now listed.",
		URL:      pgtype.Text{String: fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, m.TargetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	})

	managers, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	for _, owner := range managers.All() {
		ownerSnow, err := snowflake.Parse(owner)

		if err != nil {
			return Success{}, err
		}

		if impls.MemberOnGuild(state.Config.Servers.Main, ownerSnow) {
			if err := impls.AddRole(state.Config.Servers.Main, ownerSnow, state.Config.Roles.BotDeveloper, "Autorole due to bots owned"); err != nil {
				state.Logger.Error("Failed to add role to user", zap.Error(err), zap.String("userID", owner))
			}
		}
	}

	targetSnow, err := snowflake.Parse(m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if impls.MemberOnGuild(state.Config.Servers.Testing, targetSnow) {
		if err := impls.KickMember(state.Config.Servers.Testing, targetSnow, "Bot approved"); err != nil {
			state.Logger.Error("Failed to kick bot from testing server", zap.Error(err), zap.String("botID", m.TargetID))
		}
	}

	var clientID string

	if err := state.Pool.QueryRow(ctx, "SELECT client_id FROM bots WHERE bot_id = $1", m.TargetID).Scan(&clientID); err != nil {
		return Success{}, err
	}

	return Content(fmt.Sprintf(
		"**Invite URL:** https://discord.com/api/v10/oauth2/authorize?client_id=%s&permissions=0&scope=bot%%20applications.commands",
		clientID,
	)), nil
}

func approveServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		serverType  string
		claimedBy   *string
		lastClaimed *time.Time
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, last_claimed FROM servers WHERE server_id = $1", m.TargetID).Scan(&serverType, &claimedBy, &lastClaimed)

	if err != nil {
		return Success{}, err
	}

	if serverType != "pending" {
		return Success{}, errors.New("This server is not pending review")
	}

	if claimedBy == nil || *claimedBy == "" || lastClaimed == nil {
		return Success{}, fmt.Errorf("server `%s` is not claimed yet — claim it first", m.TargetID)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeServer, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return Success{}, err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE servers SET type = 'approved', claimed_by = NULL WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       "Server Approved",
			URL:         fmt.Sprintf("%s/servers/%s", state.Config.Sites.Frontend, m.TargetID),
			Description: fmt.Sprintf("<@!%s> has approved server `%s`", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Feedback", Value: m.Reason, Inline: impls.InlineTrue()},
				{Name: "Moderator", Value: "<@!" + h.UserID + ">", Inline: impls.InlineTrue()},
			},
			Color: impls.ColourGreen,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Success{}, err
	}

	impls.NotifyOwners(owners.All(), ptypes.Alert{
		Type:     ptypes.AlertTypeSuccess,
		Title:    "Server Approved!",
		Message:  "Your server has been approved and is now listed.",
		URL:      pgtype.Text{String: fmt.Sprintf("%s/servers/%s", state.Config.Sites.Frontend, m.TargetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	})

	return NoContent(), nil
}

func deny(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return denyServer(ctx, m, h)
	}

	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType     string
		claimedBy   *string
		owner       *string
		lastClaimed *time.Time
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, owner, last_claimed FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &owner, &lastClaimed)

	if err != nil {
		return Success{}, err
	}

	if botType != "pending" {
		return Success{}, fmt.Errorf("<@%s> is not pending review?", m.TargetID)
	}

	if claimedBy == nil || *claimedBy == "" || lastClaimed == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed? Do ``/claim`` to claim this bot first!", m.TargetID)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'denied', claimed_by = NULL WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       "Denied!",
			URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, m.TargetID),
			Description: fmt.Sprintf("<@%s> has denied <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@!" + h.UserID + ">", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Well done, young traveller at getting denied from the club!"),
			Color:  impls.ColourGreen,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	impls.NotifyOwners(owners.All(), ptypes.Alert{
		Type:     ptypes.AlertTypeError,
		Title:    "Bot Denied",
		Message:  m.Reason,
		URL:      pgtype.Text{String: fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, m.TargetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	})

	return NoContent(), nil
}

func denyServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		serverType  string
		claimedBy   *string
		lastClaimed *time.Time
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, last_claimed FROM servers WHERE server_id = $1", m.TargetID).Scan(&serverType, &claimedBy, &lastClaimed)

	if err != nil {
		return Success{}, err
	}

	if serverType != "pending" {
		return Success{}, errors.New("This server is not pending review")
	}

	if claimedBy == nil || *claimedBy == "" || lastClaimed == nil {
		return Success{}, fmt.Errorf("server `%s` is not claimed yet — claim it first", m.TargetID)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeServer, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET type = 'denied', claimed_by = NULL WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       "Server Denied",
			URL:         fmt.Sprintf("%s/servers/%s", state.Config.Sites.Frontend, m.TargetID),
			Description: fmt.Sprintf("<@%s> has denied server `%s`", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@!" + h.UserID + ">", Inline: impls.InlineTrue()},
			},
			Color: impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	impls.NotifyOwners(owners.All(), ptypes.Alert{
		Type:     ptypes.AlertTypeError,
		Title:    "Server Denied",
		Message:  m.Reason,
		URL:      pgtype.Text{String: fmt.Sprintf("%s/servers/%s", state.Config.Sites.Frontend, m.TargetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	})

	return NoContent(), nil
}

func unverify(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return unverifyServer(ctx, m, h)
	}

	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	var botType string

	if err := state.Pool.QueryRow(ctx, "SELECT type FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType); err != nil {
		return Success{}, err
	}

	if botType == "certified" {
		return Success{}, errors.New("Certified bots cannot be unverified")
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET type = 'pending', claimed_by = NULL WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__ Unverified For Futher Review!__",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Bot", Value: "<@!" + m.TargetID + ">", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Gonna be pending further review..."),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	impls.NotifyOwners(owners.All(), ptypes.Alert{
		Type:     ptypes.AlertTypeWarning,
		Title:    "Bot Unverified",
		Message:  "Your bot has been sent back for further review. " + m.Reason,
		URL:      pgtype.Text{String: fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend, m.TargetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	})

	return NoContent(), nil
}

func unverifyServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := guardServer(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	var serverType string

	if err := state.Pool.QueryRow(ctx, "SELECT type FROM servers WHERE server_id = $1", m.TargetID).Scan(&serverType); err != nil {
		return Success{}, err
	}

	if serverType == "certified" {
		return Success{}, errors.New("Certified servers cannot be unverified")
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeServer, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET type = 'pending', claimed_by = NULL WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "Server Unverified For Further Review",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Server", Value: "`" + m.TargetID + "`", Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Gonna be pending further review..."),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	impls.NotifyOwners(owners.All(), ptypes.Alert{
		Type:     ptypes.AlertTypeWarning,
		Title:    "Server Unverified",
		Message:  "Your server has been sent back for further review. " + m.Reason,
		URL:      pgtype.Text{String: fmt.Sprintf("%s/servers/%s", state.Config.Sites.Frontend, m.TargetID), Valid: true},
		Category: ptypes.AlertCategoryBotServerReviews,
	})

	return NoContent(), nil
}
