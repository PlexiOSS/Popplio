package types

import (
	"fmt"

	"popplio/badgerules"
)

// BadgeAction is the union of badge catalog operations — create/edit/delete
// what badges exist. Assigning a badge to an entity is a separate RPC
// action (AssignBadge/UnassignBadge in rpcmethod.go), not a catalog op.
type BadgeAction struct {
	List      *Unit
	RuleKinds *Unit
	Create    *BadgeUpsert
	Edit      *BadgeUpsert
	Delete    *BadgeDelete
}

type BadgeUpsert struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Icon        string           `json:"icon"`
	Color       string           `json:"color"`
	TargetTypes []string         `json:"target_types"`
	AutoRule    *badgerules.Rule `json:"auto_rule"`
	LinkedRole  bool             `json:"linked_role"`
}

type BadgeDelete struct {
	ID string `json:"id"`
}

func (a *BadgeAction) UnmarshalJSON(data []byte) error {
	*a = BadgeAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("BadgeAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "RuleKinds":
		a.RuleKinds = unitSet()
	case "Create":
		a.Create = &BadgeUpsert{}
		return decodeVariant("BadgeAction", name, payload, a.Create)
	case "Edit":
		a.Edit = &BadgeUpsert{}
		return decodeVariant("BadgeAction", name, payload, a.Edit)
	case "Delete":
		a.Delete = &BadgeDelete{}
		return decodeVariant("BadgeAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("BadgeAction", name)
	}

	return expectUnit("BadgeAction", name, payload)
}

func (a BadgeAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.RuleKinds != nil:
		return encodeUnit("RuleKinds")
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Edit != nil:
		return encodeVariant("Edit", a.Edit)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("BadgeAction: no variant set")
	}
}

type Badge struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Icon        string           `json:"icon"`
	Color       string           `json:"color"`
	TargetTypes []string         `json:"target_types"`
	AutoRule    *badgerules.Rule `json:"auto_rule,omitempty"`
	LinkedRole  bool             `json:"linked_role"`
	CreatedAt   Timestamp        `json:"created_at"`
	CreatedBy   string           `json:"created_by"`
	LastUpdated Timestamp        `json:"last_updated"`
	UpdatedBy   string           `json:"updated_by"`
}

// EntityBadge is one badge assigned to one entity — the response shape for
// GET /{target_type}/{target_id}/badges (see popplio's own, non-Arcadia,
// public badges route), not the Arcadia panel.
type EntityBadge struct {
	Badge     Badge     `json:"badge"`
	Reason    string    `json:"reason"`
	AwardedBy string    `json:"awarded_by"`
	CreatedAt Timestamp `json:"created_at"`
}
