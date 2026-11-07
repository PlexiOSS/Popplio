// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) updateSessions(ctx context.Context, q *types.QUpdateSessions) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	queries := db.New(state.Pool)

	switch {
	case q.Action.List != nil:
		rows, err := queries.ListActiveAuthChainsByUser(ctx, db.ListActiveAuthChainsByUserParams{
			UserID:       authData.UserID,
			CurrentToken: q.LoginToken,
		})

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.StaffSession, 0, len(rows))

		for _, row := range rows {
			out = append(out, types.StaffSession{
				ID:         impls.UUIDString(row.Itag),
				CreatedAt:  types.NewTimestamp(row.CreatedAt.Time),
				LastSeenAt: types.NewTimestamp(row.LastSeenAt.Time),
				IP:         ptrFromText(row.Ip),
				UserAgent:  ptrFromText(row.UserAgent),
				Current:    row.Current,
			})
		}

		return writeJSON(http.StatusOK, out), nil
	case q.Action.Revoke != nil:
		var itag pgtype.UUID

		if err := itag.Scan(q.Action.Revoke.ID); err != nil {
			return writeText(http.StatusNotFound, "Session not found"), nil
		}

		n, err := queries.DeleteAuthChainByItag(ctx, db.DeleteAuthChainByItagParams{
			Itag:   itag,
			UserID: authData.UserID,
		})

		if err != nil {
			return response{}, newError(err)
		}

		if n == 0 {
			return writeText(http.StatusNotFound, "Session not found"), nil
		}

		return writeNoContent(), nil
	case q.Action.RevokeOthers != nil:
		n, err := queries.DeleteOtherAuthChains(ctx, db.DeleteOtherAuthChainsParams{
			UserID:       authData.UserID,
			CurrentToken: q.LoginToken,
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeJSON(http.StatusOK, map[string]int64{"revoked": n}), nil
	default:
		return writeText(http.StatusBadRequest, "No action specified"), nil
	}
}
