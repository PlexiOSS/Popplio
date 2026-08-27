package rpc

import (
	"context"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
	ptypes "popplio/types"
	"popplio/votes"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Vote moderation: banning an entity from being voted for, and voiding votes
// already cast, for one entity or for every entity of a target type.
//
// Nothing here deletes a vote. Votes are voided in place with a reason and a
// timestamp, so the record of what was reset survives the reset.

// voteBanTable and its id column, per target type. Bots, servers, teams and
// packs all carry an identical vote_banned column for exactly this purpose.
func voteBanTable(targetType types.TargetType) (table, idCol string, ok bool) {
	switch targetType {
	case types.TargetTypeBot:
		return "bots", "bot_id", true
	case types.TargetTypeServer:
		return "servers", "server_id", true
	case types.TargetTypeTeam:
		return "teams", "id", true
	case types.TargetTypePack:
		return "packs", "url", true
	default:
		return "", "", false
	}
}

func voteBanSet(ctx context.Context, m *types.RPCTargetReason, h Handle, banned bool) (Success, error) {
	if err := guardEntity(ctx, h.TargetType, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	table, idCol, ok := voteBanTable(h.TargetType)

	if !ok {
		return Success{}, fmt.Errorf("vote banning does not support target type %s", h.TargetType)
	}

	query := "UPDATE " + table + " SET vote_banned = false WHERE " + idCol + " = $1"
	if banned {
		query = "UPDATE " + table + " SET vote_banned = true WHERE " + idCol + " = $1"
	}

	if _, err := state.Pool.Exec(ctx, query, m.TargetID); err != nil {
		return Success{}, err
	}

	title := "Vote Ban Removed!"
	description := fmt.Sprintf("<@%s> has removed the vote ban on <@%s>", h.UserID, m.TargetID)

	if banned {
		title = "Vote Ban Edit!"
		description = fmt.Sprintf("<@%s> has set the vote ban on <@%s>", h.UserID, m.TargetID)
	}

	err := modLogReason(
		title,
		description,
		"Remember: don't abuse our services!", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	banAlert := ptypes.Alert{
		Type:     ptypes.AlertTypeSuccess,
		Title:    "Vote Ban Removed",
		Message:  "Votes for this entity have been re-enabled.",
		Category: ptypes.AlertCategoryVotes,
	}

	if banned {
		banAlert = ptypes.Alert{
			Type:     ptypes.AlertTypeWarning,
			Title:    "Vote Banned",
			Message:  "Votes for this entity have been disabled. " + m.Reason,
			Category: ptypes.AlertCategoryVotes,
		}
	}

	notifyEntityOwners(ctx, h.TargetType, m.TargetID, banAlert)

	return NoContent(), nil
}

func voteReset(ctx context.Context, m *types.RPCTargetReason, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	_, err := state.Pool.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (single entity) reset', voided_at = NOW() WHERE target_type = $1 AND target_id = $2 AND void = FALSE",
		h.TargetType.String(), m.TargetID,
	)

	if err != nil {
		return Success{}, err
	}

	// Same cached-column problem as voteResetAll, just for one entity --
	// EntityPostVote already does exactly this recompute (it's the same
	// function a real vote triggers), so reuse it instead of hand-rolling
	// a single-row version of RecomputeApproximateVotes.
	if err := votes.EntityPostVote(ctx, state.Pool, h.TargetType.String(), m.TargetID); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__Entity Vote Reset!__",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Target ID", Value: m.TargetID, Inline: impls.InlineTrue()},
				{Name: "Target Type", Value: h.TargetType.String(), Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Sad life :("),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	notifyEntityOwners(ctx, h.TargetType, m.TargetID, ptypes.Alert{
		Type:     ptypes.AlertTypeWarning,
		Title:    "Votes Reset",
		Message:  "Your entity's votes have been reset by staff. " + m.Reason,
		Category: ptypes.AlertCategoryVotes,
	})

	return NoContent(), nil
}

func voteResetAll(ctx context.Context, m *types.RPCVoteResetAll, h Handle) (Success, error) {
	if err := checkReason(m.Reason); err != nil {
		return Success{}, err
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return Success{}, err
	}

	defer tx.Rollback(ctx)

	// Note: unlike VoteReset this has no void = FALSE filter.
	_, err = tx.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (all entities) reset', voided_at = NOW() WHERE target_type = $1 AND immutable = false",
		h.TargetType.String(),
	)

	if err != nil {
		return Success{}, err
	}

	// Voiding entity_votes rows above doesn't touch the cached
	// approximate_votes column that public listings actually sort/display
	// by -- same reasoning as VoteResetter's fix in arcadia/tasks/discord.go.
	if err := votes.RecomputeApproximateVotes(ctx, tx, h.TargetType.String()); err != nil {
		return Success{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Success{}, err
	}

	err = impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "__All Entity Votes Reset!__",
			Fields: []discord.EmbedField{
				reasonField(m.Reason),
				{Name: "Moderator", Value: "<@" + h.UserID + ">", Inline: impls.InlineTrue()},
				{Name: "Target Type", Value: h.TargetType.String(), Inline: impls.InlineTrue()},
			},
			Footer: impls.Footer("Sad life :("),
			Color:  impls.ColourRed,
		}},
	})

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

// notifyEntityOwners fetches targetID's owners and sends them alert,
// filling in URL from targetType/targetID. Best-effort throughout: a
// failure to look up owners or send the alert is logged and otherwise
// ignored, same contract as every other notification in this package.
//
// Only used for single-entity actions (vote bans, a single entity's vote
// reset) -- voteResetAll targets every entity of a type at once, which
// doesn't map to "notify this one entity's owners" the same way, so it
// intentionally isn't wired up here.
func notifyEntityOwners(ctx context.Context, targetType types.TargetType, targetID string, alert ptypes.Alert) {
	owners, err := impls.GetEntityManagers(ctx, targetType, targetID)

	if err != nil {
		state.Logger.Warn("Failed to fetch entity managers for vote notification", zap.Error(err), zap.String("targetID", targetID))
		return
	}

	urlPath := targetType.String() + "s"
	alert.URL = pgtype.Text{String: fmt.Sprintf("%s/%s/%s", state.Config.Sites.Frontend, urlPath, targetID), Valid: true}

	impls.NotifyOwners(owners.All(), alert)
}
