// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
)

const helloVersion = 5

func checkAuth(ctx context.Context, token string) (types.AuthData, error) {
	data, err := impls.CheckAuth(ctx, token)

	if err != nil {
		return types.AuthData{}, newError(err)
	}

	return data, nil
}

func checkAuthInsecure(ctx context.Context, token string) (types.AuthData, error) {
	data, err := impls.CheckAuthInsecure(ctx, token)

	if err != nil {
		return types.AuthData{}, newError(err)
	}

	return data, nil
}

func authorize(ctx context.Context, token string) (types.AuthData, perms.Set, error) {
	authData, err := checkAuth(ctx, token)

	if err != nil {
		return types.AuthData{}, perms.Set{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return types.AuthData{}, perms.Set{}, err
	}

	return authData, userPerms, nil
}

func resolvedPerms(ctx context.Context, userID string) (perms.Set, error) {
	sp, err := impls.GetUserPerms(ctx, userID)

	if err != nil {
		return perms.Set{}, newError(err)
	}

	return sp, nil
}
