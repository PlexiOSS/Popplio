// Copyright (C) 2026 NodeByte LTD

package types

import "fmt"

type DisciplinaryAction struct {
	List    *DisciplinaryList
	Issue   *DisciplinaryIssue
	Request *DisciplinaryRequest
	Approve *DisciplinaryDecision
	Deny    *DisciplinaryDecision
	Revoke  *DisciplinaryDecision
}

type DisciplinaryList struct {
	UserID *string `json:"user_id"`
	State  *string `json:"state"`
}

type DisciplinaryIssue struct {
	UserID      string `json:"user_id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Expiry      *int64 `json:"expiry"`
}

type DisciplinaryRequest struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Expiry      *int64 `json:"expiry"`
}

type DisciplinaryDecision struct {
	ID   string `json:"id"`
	Note string `json:"note"`
}

type QUpdateDisciplinaries struct {
	LoginToken string             `json:"login_token"`
	Action     DisciplinaryAction `json:"action"`
}

func (a *DisciplinaryAction) UnmarshalJSON(data []byte) error {
	*a = DisciplinaryAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("DisciplinaryAction: %w", err)
	}

	switch name {
	case "List":
		a.List = &DisciplinaryList{}
		return decodeVariant("DisciplinaryAction", name, payload, a.List)
	case "Issue":
		a.Issue = &DisciplinaryIssue{}
		return decodeVariant("DisciplinaryAction", name, payload, a.Issue)
	case "Request":
		a.Request = &DisciplinaryRequest{}
		return decodeVariant("DisciplinaryAction", name, payload, a.Request)
	case "Approve":
		a.Approve = &DisciplinaryDecision{}
		return decodeVariant("DisciplinaryAction", name, payload, a.Approve)
	case "Deny":
		a.Deny = &DisciplinaryDecision{}
		return decodeVariant("DisciplinaryAction", name, payload, a.Deny)
	case "Revoke":
		a.Revoke = &DisciplinaryDecision{}
		return decodeVariant("DisciplinaryAction", name, payload, a.Revoke)
	default:
		return errUnknownVariant("DisciplinaryAction", name)
	}
}

func (a DisciplinaryAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeVariant("List", a.List)
	case a.Issue != nil:
		return encodeVariant("Issue", a.Issue)
	case a.Request != nil:
		return encodeVariant("Request", a.Request)
	case a.Approve != nil:
		return encodeVariant("Approve", a.Approve)
	case a.Deny != nil:
		return encodeVariant("Deny", a.Deny)
	case a.Revoke != nil:
		return encodeVariant("Revoke", a.Revoke)
	default:
		return nil, fmt.Errorf("DisciplinaryAction: no variant set")
	}
}
