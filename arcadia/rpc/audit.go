package rpc

import (
	"context"

	"popplio/db"
	"popplio/state"
)

// The staff_general_logs row, which is separate from the rpc_logs row Execute
// writes for every call.
//
// rpc_logs records that a method ran and what it was given; this records what
// the world looked like before it did, which is the part that cannot be
// reconstructed afterwards.

// staffGeneralLog writes the claim/unclaim audit row.
func staffGeneralLog(ctx context.Context, userID, action, targetID string, claimedByPrev *string) error {
	data := map[string]any{
		"target_id":       targetID,
		"claimed_by_prev": claimedByPrev,
	}

	return db.New(state.Pool).InsertStaffGeneralLog(ctx, db.InsertStaffGeneralLogParams{
		UserID: userID,
		Action: action,
		Data:   data,
	})
}
