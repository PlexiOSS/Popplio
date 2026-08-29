// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/rpc"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
)

func (s *Server) executeRpc(ctx context.Context, q *types.QExecuteRpc) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	resp, rpcErr := rpc.Execute(ctx, q.Method, rpc.Handle{
		UserID:     authData.UserID,
		TargetType: q.TargetType,
	})

	if rpcErr != nil {
		return writeText(http.StatusBadRequest, rpcErr.Error()), nil
	}

	if content, ok := resp.Text(); ok {
		return writeText(http.StatusOK, content), nil
	}

	return writeNoContent(), nil
}

func (s *Server) getRpcMethods(ctx context.Context, q *types.QGetRpcMethods) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	actions := make([]types.RPCWebAction, 0, len(types.RPCMethodVariants))

	for _, name := range types.RPCMethodVariants {
		variant, err := types.EmptyRPCMethod(name)

		if err != nil {
			return response{}, newError(err)
		}

		required, known := types.RPCPermission(name)

		if q.Filtered && (!known || !userPerms.Has(required)) {
			continue
		}

		actions = append(actions, types.RPCWebAction{
			ID:                   name,
			Label:                variant.Label(),
			Description:          variant.Description(),
			SupportedTargetTypes: variant.SupportedTargetTypes(),
			Fields:               variant.Fields(),
		})
	}

	return writeJSON(http.StatusOK, actions), nil
}

func (s *Server) getRpcLogEntries(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !userPerms.Has(perms.StaffViewAuditLogs) {
		return writeText(http.StatusForbidden, "You do not have permission to view rpc logs [view_audit_logs]"), nil
	}

	entries, err := db.New(state.Pool).ListRPCLogs(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	log := make([]types.RPCLogEntry, 0, len(entries))

	for _, entry := range entries {
		data, err := json.Marshal(entry.Data)

		if err != nil {
			return response{}, newError(err)
		}

		log = append(log, types.RPCLogEntry{
			ID:        impls.UUIDString(entry.ID),
			UserID:    entry.UserID,
			Method:    entry.Method,
			Data:      data,
			State:     entry.State,
			CreatedAt: types.NewTimestamp(entry.CreatedAt.Time),
		})
	}

	return writeJSON(http.StatusOK, log), nil
}
