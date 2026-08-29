// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"
)

func (s *Server) hello(ctx context.Context, q *types.QHello) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if q.Version != helloVersion {
		return writeText(http.StatusBadRequest, "Invalid version"), nil
	}

	staffMember, err := impls.GetStaffMember(ctx, authData.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, types.Hello{
		InstanceConfig: types.InstanceConfig{
			Description: instanceDescription(),
			Warnings:    []string{},
		},
		AuthData:    authData,
		StaffMember: staffMember,
		CoreConstants: types.CoreConstants{
			FrontendURL:    state.Config.Sites.Frontend,
			InfernoplexURL: state.Config.Sites.Infernoplex,
			PopplioURL:     state.Config.Sites.API,
			Servers:        serverIDs(),
		},
		TargetTypes: types.TargetTypeVariants,
	}), nil
}

func (s *Server) baseAnalytics(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	queries := db.New(state.Pool)

	botTypeRows, err := queries.CountBotsByType(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	botCounts := make(map[string]int64, len(botTypeRows))

	for _, row := range botTypeRows {
		botCounts[row.Method] = row.Count
	}

	serverTypeRows, err := queries.CountServersByType(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	serverCounts := make(map[string]int64, len(serverTypeRows))

	for _, row := range serverTypeRows {
		serverCounts[row.Method] = row.Count
	}

	ticketRows, err := queries.CountTicketsByOpen(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	ticketCounts := make(map[string]int64, len(ticketRows))

	for _, row := range ticketRows {
		if row.Open {
			ticketCounts["open"] = row.Count
		} else {
			ticketCounts["closed"] = row.Count
		}
	}

	totalUsers, err := queries.CountUsers(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, types.BaseAnalytics{
		BotCounts:       botCounts,
		ServerCounts:    serverCounts,
		TicketCounts:    ticketCounts,
		TotalUsers:      totalUsers,
		ChangelogsCount: 0,
	}), nil
}
