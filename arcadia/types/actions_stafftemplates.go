package types

import "fmt"

// StaffTemplateAction is the union of staff-template catalog operations —
// pre-built answers staff pick from when approving/denying/etc, for both
// bot and server reviews (see StaffTemplate.EntityType). Same shape as
// BadgeAction (actions_badges.go), the closest existing precedent for a
// simple staff-managed catalog with no rank/hierarchy concerns.
type StaffTemplateAction struct {
	List   *Unit
	Create *StaffTemplateUpsert
	Edit   *StaffTemplateUpsert
	Delete *StaffTemplateDelete
}

type StaffTemplateUpsert struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	// "bot" or "server" — which review queue this template shows up in.
	EntityType string `json:"entity_type"`
}

type StaffTemplateDelete struct {
	ID string `json:"id"`
}

func (a *StaffTemplateAction) UnmarshalJSON(data []byte) error {
	*a = StaffTemplateAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("StaffTemplateAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "Create":
		a.Create = &StaffTemplateUpsert{}
		return decodeVariant("StaffTemplateAction", name, payload, a.Create)
	case "Edit":
		a.Edit = &StaffTemplateUpsert{}
		return decodeVariant("StaffTemplateAction", name, payload, a.Edit)
	case "Delete":
		a.Delete = &StaffTemplateDelete{}
		return decodeVariant("StaffTemplateAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("StaffTemplateAction", name)
	}

	return expectUnit("StaffTemplateAction", name, payload)
}

func (a StaffTemplateAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Edit != nil:
		return encodeVariant("Edit", a.Edit)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("StaffTemplateAction: no variant set")
	}
}
