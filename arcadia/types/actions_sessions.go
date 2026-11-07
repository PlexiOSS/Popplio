// Copyright (C) 2026 NodeByte LTD

package types

import "fmt"

type SessionAction struct {
	List         *SessionList
	Revoke       *SessionRevoke
	RevokeOthers *SessionList
}

type SessionList struct{}

type SessionRevoke struct {
	ID string `json:"id"`
}

type StaffSession struct {
	ID         string    `json:"id"`
	CreatedAt  Timestamp `json:"created_at"`
	LastSeenAt Timestamp `json:"last_seen_at"`
	IP         *string   `json:"ip"`
	UserAgent  *string   `json:"user_agent"`
	Current    bool      `json:"current"`
}

type QUpdateSessions struct {
	LoginToken string        `json:"login_token"`
	Action     SessionAction `json:"action"`
}

func (a *SessionAction) UnmarshalJSON(data []byte) error {
	*a = SessionAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("SessionAction: %w", err)
	}

	switch name {
	case "List":
		a.List = &SessionList{}
		return decodeVariant("SessionAction", name, payload, a.List)
	case "Revoke":
		a.Revoke = &SessionRevoke{}
		return decodeVariant("SessionAction", name, payload, a.Revoke)
	case "RevokeOthers":
		a.RevokeOthers = &SessionList{}
		return decodeVariant("SessionAction", name, payload, a.RevokeOthers)
	default:
		return errUnknownVariant("SessionAction", name)
	}
}

func (a SessionAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeVariant("List", a.List)
	case a.Revoke != nil:
		return encodeVariant("Revoke", a.Revoke)
	case a.RevokeOthers != nil:
		return encodeVariant("RevokeOthers", a.RevokeOthers)
	default:
		return nil, fmt.Errorf("SessionAction: no variant set")
	}
}
