// Copyright (C) 2026 NodeByte LTD

package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"popplio/constants"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/urlutil"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/uapi"
)

type PermissionCheck struct {
	NeededPermission func(d uapi.Route, r *http.Request, authData uapi.AuthData) (*perms.Perm, error)
	GetTarget        func(d uapi.Route, r *http.Request, authData uapi.AuthData) (targetType string, targetId string)
}

const (
	SESSION_EXPIRY       = 60 * 30
	PERMISSION_CHECK_KEY = "permissionCheck"
)

const (
	TargetTypeUser   = "user"
	TargetTypeBot    = "bot"
	TargetTypeServer = "server"
	TargetTypeTeam   = "team"
)

// Returns all possible auth types
func GetAllAuthTypes() []uapi.AuthType {
	return []uapi.AuthType{
		{
			Type: TargetTypeUser,
		},
		{
			Type: TargetTypeBot,
		},
		{
			Type: TargetTypeServer,
		},
		{
			Type: TargetTypeTeam,
		},
	}
}

type DefaultResponder struct{}

func (d DefaultResponder) New(err string, ctx map[string]string) any {
	return types.ApiError{
		Message: err,
		Context: ctx,
	}
}

func PermLimits(d uapi.AuthData) []string {
	if !d.Authorized {
		return []string{}
	}

	permLimits, ok := d.Data["perm_limits"].([]string)

	if !ok {
		panic("Could not assert perm limits as []string")
	}

	return permLimits
}

func EntityPerms(d uapi.AuthData) []string {
	if !d.Authorized {
		return []string{}
	}

	permLimits, ok := d.Data["entity_perms"].([]string)

	if !ok {
		panic("Could not assert perm limits as []string")
	}

	return permLimits
}

func Authorize(r uapi.Route, req *http.Request) (uapi.AuthData, uapi.HttpResponse, bool) {
	if len(r.Auth) == 0 {
		return uapi.AuthData{}, uapi.HttpResponse{}, true
	}

	authHeader := req.Header.Get("Authorization")

	if len(r.Auth) > 0 && authHeader == "" && !r.AuthOptional {
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusUnauthorized), false
	}

	authData := uapi.AuthData{}

	q := db.New(state.Pool)

	err := q.DeleteExpiredSessions(state.Context)

	if err != nil {
		state.Logger.Error("Failed to delete expired web API tokens [db delete]", zap.Error(err))
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
	}

	var authPrefix string
	parts := strings.SplitN(authHeader, " ", 2)

	if len(parts) == 2 {
		authPrefix = strings.ToLower(parts[0])
		authHeader = parts[1]
	}

	sess, err := q.GetSessionByToken(state.Context, authHeader)
	sessId, targetId, targetType, permLimits := sess.ID, sess.TargetID, sess.TargetType, sess.PermLimits

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid session token"},
			Headers: map[string]string{
				"X-Session-Invalid": "true",
			},
		}, false
	}

	if err != nil {
		state.Logger.Error("Failed to fetch session [db fetch]", zap.Error(err))
		return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
	}

	if len(permLimits) == 0 {
		permLimits = []string{}
	}

	if authPrefix != "" && authPrefix != targetType {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Invalid authorization prefix, expected " + authPrefix + " but got " + targetType},
			Headers: map[string]string{
				"X-Session-Invalid": "true",
			},
		}, false
	}

	state.Logger.Info("All auth types", zap.Any("auth", r.Auth))
	for _, auth := range r.Auth {

		if authData.Authorized {
			break
		}

		if targetType != auth.Type {
			state.Logger.Info("Ignoring auth type", zap.String("authType", auth.Type), zap.String("targetType", targetType))
			continue
		}

		switch auth.Type {
		case TargetTypeUser:
			banStatus, err := q.GetUserBanStatus(state.Context, targetId)
			banned, bugHunter := banStatus.Banned, banStatus.BugHunters

			if err != nil {
				state.Logger.Error("Failed to fetch user associated with session [db fetch]", zap.Error(err), zap.String("userID", targetId))
				return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
			}

			if urlutil.DifferentHost(req.Header.Get("Origin"), state.Config.Sites.Frontend) &&
				!bugHunter && !perms.IsConfigOwner(targetId) {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "This environment is limited to Bug Hunters."},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeUser,
				ID:         targetId,
				Authorized: true,
				Banned:     banned,
			}
		case TargetTypeBot:
			count, err := q.CountBotByID(state.Context, targetId)

			if err != nil {
				state.Logger.Error("Failed to fetch bot count associated with session [db fetch]", zap.Error(err), zap.String("botID", targetId))
				return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
			}

			if count == 0 {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusNotFound,
					Json:   types.ApiError{Message: "The bot associated with this session could not be found?"},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeBot,
				ID:         targetId,
				Authorized: true,
			}
		case TargetTypeServer:
			count, err := q.CountServerByID(state.Context, targetId)

			if err != nil {
				state.Logger.Error("Failed to fetch server count associated with session [db fetch]", zap.Error(err), zap.String("serverID", targetId))
				return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
			}

			if count == 0 {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusNotFound,
					Json:   types.ApiError{Message: "The server associated with this session could not be found?"},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeServer,
				ID:         targetId,
				Authorized: true,
			}
		case TargetTypeTeam:
			count, err := q.CountTeamByID(state.Context, targetId)

			if err != nil {
				state.Logger.Error("Failed to fetch team count associated with session [db fetch]", zap.Error(err), zap.String("teamID", targetId))
				return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
			}

			if count == 0 {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusNotFound,
					Json:   types.ApiError{Message: "The team associated with this session could not be found?"},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}

			authData = uapi.AuthData{
				TargetType: TargetTypeTeam,
				ID:         targetId,
				Authorized: true,
			}
		}

		if authData.Authorized {
			if auth.URLVar != "" {
				state.Logger.Info("Checking URL variable against user ID from auth token", zap.String("URLVar", auth.URLVar))
				gotUserId := chi.URLParam(req, auth.URLVar)
				if gotUserId != targetId {
					return uapi.AuthData{}, uapi.HttpResponse{
						Status: http.StatusForbidden,
						Json:   types.ApiError{Message: "You are not authorized to perform this action (URLVar does not match auth token)"},
						Headers: map[string]string{
							"X-Session-Invalid": "true",
						},
					}, false
				}
			}

			if authData.Banned && auth.AllowedScope != "ban_exempt" {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "You are banned from the list. If you think this is a mistake, please contact support."},
					Headers: map[string]string{
						"X-Session-Invalid": "true",
					},
				}, false
			}
		}
	}

	authData.Data = map[string]any{
		"session_id":  sessId,
		"perm_limits": permLimits,
	}

	if !authData.Authorized && !r.AuthOptional {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusUnauthorized,
			Json:   types.ApiError{Message: "Authentication failed due to lack of target of type support? [!authData.Authorized && !r.AuthOptional]"},
		}, false
	}

	state.Logger.Info("AuthData", zap.Any("authData", authData))

	pc, ok := r.ExtData[PERMISSION_CHECK_KEY]

	if !ok {
		return uapi.AuthData{}, uapi.HttpResponse{
			Status: http.StatusInternalServerError,
			Json:   types.ApiError{Message: "Internal server error: permissionCheck not found in route.ExtData"},
		}, false
	}

	permCheck, ok := pc.(PermissionCheck)

	if ok {
		if permCheck.NeededPermission == nil {
			return uapi.AuthData{}, uapi.HttpResponse{
				Status: http.StatusInternalServerError,
				Json:   types.ApiError{Message: "Internal error: NeededPermission function is nil"},
			}, false
		}

		neededPerm, err := permCheck.NeededPermission(r, req, authData)

		if err != nil {
			state.Logger.Error("Failed to resolve needed permission for authorization", zap.Error(err), zap.String("opId", r.OpId))
			return uapi.AuthData{}, uapi.DefaultResponse(http.StatusInternalServerError), false
		}

		if neededPerm != nil {
			if permCheck.GetTarget == nil {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusInternalServerError,
					Json:   types.ApiError{Message: "Internal error: GetTarget function is nil"},
				}, false
			}

			targetTypeOfEntity, targetIdOfEntity := permCheck.GetTarget(r, req, authData)

			if targetTypeOfEntity == "" || targetIdOfEntity == "" {
				return uapi.AuthData{}, uapi.HttpResponse{
					Status: http.StatusBadRequest,
					Json:   types.ApiError{Message: "Internal error: Both target_id and target_type must be specified in the route.ExtData[PERMISSION_CHECK_KEY]"},
				}, false
			}

			err = AuthzEntityPermissionCheck(
				req.Context(),
				authData,
				targetTypeOfEntity,
				targetIdOfEntity,
				*neededPerm,
			)

			if err != nil {
				return authData, uapi.HttpResponse{
					Status: http.StatusForbidden,
					Json:   types.ApiError{Message: "Entity permission checks failed: " + err.Error()},
				}, false
			}
		}
	}

	return authData, uapi.HttpResponse{}, true
}

func Setup() {
	uapi.SetupState(uapi.UAPIState{
		Logger:    state.Logger,
		Authorize: Authorize,
		AuthTypeMap: func() map[string]string {
			return map[string]string{
				TargetTypeUser:   "User",
				TargetTypeBot:    "Bot",
				TargetTypeServer: TargetTypeServer,
				TargetTypeTeam:   TargetTypeTeam,
			}
		}(),
		Context: state.Context,
		Constants: &uapi.UAPIConstants{
			ResourceNotFound:    constants.ResourceNotFound,
			BadRequest:          constants.BadRequest,
			Forbidden:           constants.Forbidden,
			Unauthorized:        constants.Unauthorized,
			InternalServerError: constants.InternalServerError,
			MethodNotAllowed:    constants.MethodNotAllowed,
			BodyRequired:        constants.BodyRequired,
		},
		DefaultResponder: DefaultResponder{},
		BaseSanityCheck: func(r uapi.Route) error {
			if len(r.Auth) > 0 {
				if _, ok := r.ExtData[PERMISSION_CHECK_KEY]; !ok {
					return fmt.Errorf("%s not found in route.ExtData [%s]", PERMISSION_CHECK_KEY, r.OpId)
				}
			}

			return nil
		},
	})
}
