// Copyright (C) 2026 NodeByte LTD

package types

type LinkedRoleBadge struct {
	ID          string `json:"id" description:"The badge ID, also the Discord metadata key"`
	Name        string `json:"name" description:"The badge name"`
	Description string `json:"description" description:"The badge description"`
}

type LinkedRolesMeta struct {
	Enabled  bool              `json:"enabled" description:"Whether linked roles are configured"`
	ClientID string            `json:"client_id" description:"The Discord application to authorize"`
	Badges   []LinkedRoleBadge `json:"badges" description:"The badges Discord servers can use as linked-role requirements"`
}

type LinkedRolesConnect struct {
	Code        string `json:"code" validate:"required" description:"The OAuth2 code Discord returned"`
	RedirectURI string `json:"redirect_uri" validate:"required" description:"The redirect URI used for the authorization"`
}

type LinkedRolesResult struct {
	UserID   string   `json:"user_id" description:"The connected Discord user ID"`
	Username string   `json:"username" description:"The connected Discord username"`
	Badges   []string `json:"badges" description:"The names of the linked-role badges this user has"`
}
