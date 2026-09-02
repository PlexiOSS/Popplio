// Copyright (C) 2026 NodeByte LTD

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"popplio/perms"
	"popplio/teams"

	"github.com/PlexiOSS/Keel/uapi"
)

var (
	ErrCrossEntityNotSupported     = errors.New("cross entity actions with this auth type are not supported")
	ErrUsersCannotModifyOtherUsers = errors.New("users cannot modify other users")
	ErrMissingPermission           = errors.New("missing permission")
	ErrInvalidTargetType           = errors.New("invalid target type")
)

func AuthzEntityPermissionCheck(
	ctx context.Context,
	authData uapi.AuthData,
	targetType string,
	targetId string,
	perm perms.Perm,
) error {
	if _, ok := uapi.State.AuthTypeMap[targetType]; !ok {
		return ErrInvalidTargetType
	}

	permLimits := PermLimits(authData)

	if len(permLimits) > 0 {
		if !perms.Entity.ResolveStrings(permLimits).Has(perm) {
			return fmt.Errorf("%w: %s", ErrMissingPermission, perm)
		}
	}

	if targetType == authData.TargetType && targetId == authData.ID {
		return nil
	}

	switch authData.TargetType {
	case TargetTypeUser:
		if targetType == "user" {
			return ErrUsersCannotModifyOtherUsers
		}

		entityPerms, err := teams.GetEntityPerms(ctx, authData.ID, targetType, targetId)

		if err != nil {
			return err
		}

		if !entityPerms.Has(perm) {
			return fmt.Errorf("%w: %s", ErrMissingPermission, perm)
		}

		return nil
	default:
		return ErrCrossEntityNotSupported
	}
}

func Needs(p perms.Perm) func(uapi.Route, *http.Request, uapi.AuthData) (*perms.Perm, error) {
	return func(uapi.Route, *http.Request, uapi.AuthData) (*perms.Perm, error) {
		return &p, nil
	}
}
