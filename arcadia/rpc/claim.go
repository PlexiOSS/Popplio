package rpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

func claim(ctx context.Context, m *types.RPCClaim, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return claimServer(ctx, m, h)
	}

	var (
		botType   string
		claimedBy *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy)

	if err != nil {
		return Success{}, err
	}

	if botType != "pending" {
		return Success{}, errors.New("This bot is not pending review")
	}

	if botType == "testbot" {
		return Success{}, errors.New("This bot is a test bot")
	}

	if !m.Force && claimedBy != nil {
		return Success{}, fmt.Errorf("This bot is already claimed by <@%s>", *claimedBy)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET last_claimed = NOW(), claimed_by = $1 WHERE bot_id = $2", h.UserID, m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "claimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Claimed!",
			Description: fmt.Sprintf("<@%s> has claimed <@%s>", h.UserID, m.TargetID),
			Color:       impls.ColourBlurple,
			Fields: []discord.EmbedField{
				{Name: "Force Claim", Value: strconv.FormatBool(m.Force), Inline: impls.InlineFalse()},
			},
			Footer: impls.Footer("This is completely normal, don't worry!"),
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

/**
 * Server Claiming
 *
 * Server claiming is a bit different from bot claiming, as servers are not owned by a single user.
 * Instead, they are owned by a group of users, and any of them can claim the server.
 * The claim is still stored in the database, but it is not tied to a single user.
 * Instead, the claim is tied to the server itself, and any of the owners can unclaim it.
 */
func claimServer(ctx context.Context, m *types.RPCClaim, h Handle) (Success, error) {
	var (
		serverType string
		claimedBy  *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by FROM servers WHERE server_id = $1", m.TargetID).Scan(&serverType, &claimedBy)

	if err != nil {
		return Success{}, err
	}

	if serverType != "pending" {
		return Success{}, errors.New("This server is not pending review")
	}

	if !m.Force && claimedBy != nil {
		return Success{}, fmt.Errorf("This server is already claimed by <@%s>", *claimedBy)
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeServer, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET last_claimed = NOW(), claimed_by = $1 WHERE server_id = $2", h.UserID, m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "claimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       "Server Claimed",
			Description: fmt.Sprintf("<@%s> has claimed server `%s`", h.UserID, m.TargetID),
			Color:       impls.ColourBlurple,
			Fields: []discord.EmbedField{
				{Name: "Force Claim", Value: strconv.FormatBool(m.Force), Inline: impls.InlineFalse()},
			},
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

/**
 * Unclaiming
 *
 * Unclaiming is the opposite of claiming, and can be done by any staff member.
 * It is not tied to a single user, and can be done by any staff member.
 * The unclaim is stored in the database, and any of the owners can claim it again.
 */
func unclaim(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if h.TargetType == types.TargetTypeServer {
		return unclaimServer(ctx, m, h)
	}

	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		botType   string
		claimedBy *string
		owner     *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by, owner FROM bots WHERE bot_id = $1", m.TargetID).Scan(&botType, &claimedBy, &owner)

	if err != nil {
		return Success{}, err
	}

	if botType == "testbot" {
		return Success{}, errors.New("This bot is a test bot")
	}

	if botType != "pending" {
		return Success{}, errors.New("This bot is not pending review")
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeBot, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if claimedBy == nil {
		return Success{}, fmt.Errorf("<@%s> is not claimed", m.TargetID)
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE bot_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "unclaimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       " Unclaimed!",
			Description: fmt.Sprintf("<@%s> has unclaimed <@%s>", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineFalse()},
			},
			Footer: impls.Footer("This is completely normal, don't worry!"),
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

/**
 * Unclaiming Servers
 *
 * Unclaiming servers is the opposite of claiming, and can be done by any staff member.
 * It is not tied to a single user, and can be done by any staff member.
 * The unclaim is stored in the database, and any of the owners can claim it again.
 */
func unclaimServer(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	var (
		serverType string
		claimedBy  *string
	)

	err := state.Pool.QueryRow(ctx, "SELECT type, claimed_by FROM servers WHERE server_id = $1", m.TargetID).Scan(&serverType, &claimedBy)

	if err != nil {
		return Success{}, err
	}

	if serverType != "pending" {
		return Success{}, errors.New("This server is not pending review")
	}

	owners, err := impls.GetEntityManagers(ctx, types.TargetTypeServer, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if claimedBy == nil {
		return Success{}, fmt.Errorf("server `%s` is not claimed", m.TargetID)
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE servers SET claimed_by = NULL, type = 'pending' WHERE server_id = $1", m.TargetID); err != nil {
		return Success{}, err
	}

	if err := staffGeneralLog(ctx, h.UserID, "unclaimed", m.TargetID, claimedBy); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Content: owners.MentionUsers(),
		Embeds: []discord.Embed{{
			Title:       "Server Unclaimed",
			Description: fmt.Sprintf("<@%s> has unclaimed server `%s`", h.UserID, m.TargetID),
			Fields: []discord.EmbedField{
				{Name: "Reason", Value: m.Reason, Inline: impls.InlineFalse()},
			},
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
