// Package rpc is the shared action layer behind both the staff panel and the
// staff Discord bot. A staff member approving a bot from the web panel and one
// approving it with a slash command run the same handler, the same permission
// check, the same rate limit, the same audit row and the same mod-log embed.
//
// # Layout
//
// core.go holds the pipeline every action goes through (Execute) and the guards
// they share; dispatch.go maps a method to its handler; the handlers themselves
// are grouped by what they act on and by the permission that gates them:
//
//	review.go       claim, unclaim, approve, deny, unverify   review_entities
//	certify.go      certification                             certify_entities
//	transfer.go     ownership changes                         transfer_bots
//	forceremove.go  deleting a bot/server/pack outright        force_remove_entities
//	premium.go      premium                                   manage_premium
//	votes.go        vote bans and vote resets                 manage_votes, ban_voters
//	apps.go         application bans                          ban_app_users
//	audit.go        the staff_general_logs row
//
// # Embeds are reproduced verbatim
//
// Every mod-log embed in this package — titles (leading spaces included),
// descriptions, field names, inline flags, footers and colours — is reproduced
// exactly from the Rust original. The mod-log channel is read by humans and
// several titles have odd leading spaces that are intentional by accident. See
// arcadia/CONFORMANCE.md before tidying one up.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

// Handle carries the caller identity and target type for one RPC execution.
type Handle struct {
	UserID     string
	TargetType types.TargetType
}

// Success is either an empty result (204 from the panel) or a text payload
// (200 text/plain from the panel).
type Success struct {
	content *string
}

// NoContent is a successful result with no body.
func NoContent() Success { return Success{} }

// Content is a successful result carrying raw text.
func Content(s string) Success { return Success{content: &s} }

// Text returns the payload and whether one was set.
func (s Success) Text() (string, bool) {
	if s.content == nil {
		return "", false
	}

	return *s.content, true
}

// Execute runs the full RPC pipeline: target-type check, permission check, audit
// row, rate limit, handler, then the audit row's final state.
//
// The order matters and is reproduced exactly. In particular the audit row is
// inserted BEFORE the rate limit is counted, so the current call counts towards
// its own limit: the effective allowance is 5 calls per rolling 7 minutes and the
// 6th is rejected.
func Execute(ctx context.Context, method types.RPCMethod, h Handle) (Success, error) {
	if !supportsTargetType(method, h.TargetType) {
		return Success{}, errors.New("This method does not support this target type yet")
	}

	userPerms, err := impls.GetUserPerms(ctx, h.UserID)

	if err != nil {
		return Success{}, err
	}

	required, ok := method.Permission()

	if !ok {
		return Success{}, fmt.Errorf("Unknown method %s", method.Name())
	}

	if !userPerms.Has(required) {
		return Success{}, fmt.Errorf("You need the %s permission to use %s", required, method.Name())
	}

	data, err := json.Marshal(method)

	if err != nil {
		return Success{}, err
	}

	var dataMap map[string]any

	if err := json.Unmarshal(data, &dataMap); err != nil {
		return Success{}, err
	}

	q := db.New(state.Pool)

	logID, err := q.InsertRPCLog(ctx, db.InsertRPCLogParams{
		Method: method.Name(),
		UserID: h.UserID,
		Data:   dataMap,
	})

	if err != nil {
		return Success{}, err
	}

	count, err := q.CountRecentRPCLogs(ctx, h.UserID)

	if err != nil {
		return Success{}, errors.New("Failed to get ratelimit count")
	}

	if count > 5 {
		err = q.DeleteAuthChainByUserID(ctx, h.UserID)

		if err != nil {
			return Success{}, errors.New("Failed to reset user token")
		}

		return Success{}, errors.New("Rate limit exceeded. Wait 5-10 minutes and try again?")
	}

	resp, handlerErr := handleMethod(ctx, method, h)

	logState := "success"
	if handlerErr != nil {
		logState = handlerErr.Error()
	}

	if err := q.UpdateRPCLogState(ctx, db.UpdateRPCLogStateParams{State: logState, ID: logID}); err != nil {
		return Success{}, err
	}

	return resp, handlerErr
}

func supportsTargetType(method types.RPCMethod, target types.TargetType) bool {
	for _, t := range method.SupportedTargetTypes() {
		if t == target {
			return true
		}
	}

	return false
}

// checkReason enforces MAX_REASON_LENGTH. The message is user-visible.
func checkReason(reason string) error {
	if len(reason) > types.MaxReasonLength {
		return fmt.Errorf("Reason must be lower than/equal to %d characters", types.MaxReasonLength)
	}

	return nil
}

// entityExists is the "SELECT COUNT(*)" existence guard several methods run.
func entityExists(count int64, err error, targetID string) error {
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("%q does not exist", targetID)
	}

	return nil
}

func botExists(ctx context.Context, targetID string) error {
	count, err := db.New(state.Pool).CountBotByID(ctx, targetID)
	return entityExists(count, err, targetID)
}

func userExists(ctx context.Context, targetID string) error {
	exists, err := db.New(state.Pool).UserExists(ctx, targetID)

	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%q does not exist", targetID)
	}

	return nil
}

func serverExists(ctx context.Context, targetID string) error {
	count, err := db.New(state.Pool).CountServerByID(ctx, targetID)
	return entityExists(count, err, targetID)
}

func teamExists(ctx context.Context, targetID string) error {
	count, err := db.New(state.Pool).CountTeamByID(ctx, targetID)
	return entityExists(count, err, targetID)
}

func packExists(ctx context.Context, targetID string) error {
	count, err := db.New(state.Pool).CountPackByURL(ctx, targetID)
	return entityExists(count, err, targetID)
}

// guardEntity is guardBot/guardUser generalized across every target type a
// method might be extended to support — used by handlers whose
// SupportedTargetTypes covers more than one entity kind.
func guardEntity(ctx context.Context, targetType types.TargetType, targetID, reason string) error {
	if err := checkReason(reason); err != nil {
		return err
	}

	switch targetType {
	case types.TargetTypeBot:
		return botExists(ctx, targetID)
	case types.TargetTypeServer:
		return serverExists(ctx, targetID)
	case types.TargetTypeTeam:
		return teamExists(ctx, targetID)
	case types.TargetTypePack:
		return packExists(ctx, targetID)
	case types.TargetTypeUser:
		return userExists(ctx, targetID)
	default:
		return fmt.Errorf("unsupported target type %s", targetType)
	}
}
