// Copyright (C) 2026 NodeByte LTD

package types

import "fmt"

type AppPositionQuestion struct {
	ID          string `json:"id"`
	Question    string `json:"question"`
	Paragraph   string `json:"paragraph"`
	Placeholder string `json:"placeholder"`
	Short       bool   `json:"short"`
}

type AppPositionBehaviour struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	LockedQuestions []string `json:"locked_questions"`
}

type AppPosition struct {
	ID                 string                `json:"id"`
	Name               string                `json:"name"`
	Info               string                `json:"info"`
	Tags               []string              `json:"tags"`
	Questions          []AppPositionQuestion `json:"questions"`
	Hidden             bool                  `json:"hidden"`
	Closed             bool                  `json:"closed"`
	RequiresMainServer bool                  `json:"requires_main_server"`
	CooldownSecs       int64                 `json:"cooldown_secs"`
	ApprovalNote       string                `json:"approval_note"`
	HelpURL            string                `json:"help_url"`
	HelpLabel          string                `json:"help_label"`
	Behaviour          *AppPositionBehaviour `json:"behaviour"`
	PendingApps        int64                 `json:"pending_apps"`
	UpdatedAt          Timestamp             `json:"updated_at"`
}

type AppPositionInput struct {
	ID                 string                `json:"id"`
	Name               string                `json:"name"`
	Info               string                `json:"info"`
	Tags               []string              `json:"tags"`
	Questions          []AppPositionQuestion `json:"questions"`
	Hidden             bool                  `json:"hidden"`
	Closed             bool                  `json:"closed"`
	RequiresMainServer bool                  `json:"requires_main_server"`
	CooldownSecs       int64                 `json:"cooldown_secs"`
	ApprovalNote       string                `json:"approval_note"`
	HelpURL            string                `json:"help_url"`
	HelpLabel          string                `json:"help_label"`
}

type AppPositionList struct{}

type AppPositionRef struct {
	ID string `json:"id"`
}

type AppPositionReorder struct {
	IDs []string `json:"ids"`
}

type AppPositionAction struct {
	List    *AppPositionList
	Create  *AppPositionInput
	Update  *AppPositionInput
	Delete  *AppPositionRef
	Reorder *AppPositionReorder
}

type QUpdateAppPositions struct {
	LoginToken string            `json:"login_token"`
	Action     AppPositionAction `json:"action"`
}

func (a *AppPositionAction) UnmarshalJSON(data []byte) error {
	*a = AppPositionAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("AppPositionAction: %w", err)
	}

	switch name {
	case "List":
		a.List = &AppPositionList{}
		return decodeVariant("AppPositionAction", name, payload, a.List)
	case "Create":
		a.Create = &AppPositionInput{}
		return decodeVariant("AppPositionAction", name, payload, a.Create)
	case "Update":
		a.Update = &AppPositionInput{}
		return decodeVariant("AppPositionAction", name, payload, a.Update)
	case "Delete":
		a.Delete = &AppPositionRef{}
		return decodeVariant("AppPositionAction", name, payload, a.Delete)
	case "Reorder":
		a.Reorder = &AppPositionReorder{}
		return decodeVariant("AppPositionAction", name, payload, a.Reorder)
	default:
		return errUnknownVariant("AppPositionAction", name)
	}
}

func (a AppPositionAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeVariant("List", a.List)
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Update != nil:
		return encodeVariant("Update", a.Update)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	case a.Reorder != nil:
		return encodeVariant("Reorder", a.Reorder)
	default:
		return nil, fmt.Errorf("AppPositionAction: no variant set")
	}
}
