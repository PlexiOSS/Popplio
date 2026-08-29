package rpc

import (
	"context"
	"errors"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Moving a bot to a different owner, gated by transfer_bots.
//
// A bot is owned by either a user or a team and never both, so there are two
// handlers rather than one: each refuses the case the other exists for.

func transferOwnershipUser(ctx context.Context, m *types.RPCBotTransferOwnershipUser, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	q := db.New(state.Pool)

	teamOwner, err := q.GetBotTeamOwner(ctx, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if teamOwner.Valid {
		return Success{}, fmt.Errorf("%q is in a team. Please use BotTransferOwnershipTeam", m.TargetID)
	}

	if err := q.UpdateBotOwner(ctx, db.UpdateBotOwnerParams{
		BotID: m.TargetID,
		Owner: pgtype.Text{String: m.NewOwner, Valid: true},
	}); err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Ownership Force Update!",
		fmt.Sprintf("<@%s> has force-updated the ownership of <@%s> to <@%s>", h.UserID, m.TargetID, m.NewOwner),
		"Contact support if you think this is a mistake", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}

func transferOwnershipTeam(ctx context.Context, m *types.RPCBotTransferOwnershipTeam, h Handle) (Success, error) {
	if err := guardBot(ctx, m.TargetID, m.Reason); err != nil {
		return Success{}, err
	}

	teamID, err := uuid.Parse(m.NewTeam)

	if err != nil {
		return Success{}, errors.New("Invalid team ID")
	}

	q := db.New(state.Pool)

	teamOwner, err := q.GetBotTeamOwner(ctx, m.TargetID)

	if err != nil {
		return Success{}, err
	}

	if !teamOwner.Valid {
		return Success{}, fmt.Errorf("%q is not in a team. Please use TransferOwnership", m.TargetID)
	}

	if err := q.SetBotTeamOwnerDirect(ctx, db.SetBotTeamOwnerDirectParams{
		BotID:     m.TargetID,
		TeamOwner: pgtype.UUID{Bytes: teamID, Valid: true},
	}); err != nil {
		return Success{}, err
	}

	err = modLogReason(
		"Ownership Force Update!",
		fmt.Sprintf("<@%s> has force-updated the ownership of <@%s> to team %s", h.UserID, m.TargetID, teamID),
		"Contact support if you think this is a mistake", impls.ColourRed, m.Reason)

	if err != nil {
		return Success{}, err
	}

	return NoContent(), nil
}
