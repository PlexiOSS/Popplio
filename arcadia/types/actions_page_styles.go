// Copyright (C) 2026 NodeByte LTD

package types

import "fmt"

type PageStyleAction struct {
	ListPending *Unit
	Approve     *PageImageDecision
	Reject      *PageImageDecision
}

type PageImageDecision struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Version    string `json:"version"`
	Reason     string `json:"reason"`
}

type PendingPageImage struct {
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	Name          string    `json:"name"`
	Avatar        string    `json:"avatar"`
	Version       string    `json:"version"`
	ImageURL      string    `json:"image_url"`
	ReplacesURL   *string   `json:"replaces_url"`
	Dim           int       `json:"dim"`
	Blur          int       `json:"blur"`
	ImagePosition string    `json:"image_position"`
	PendingSince  Timestamp `json:"pending_since"`
}

func (a *PageStyleAction) UnmarshalJSON(data []byte) error {
	*a = PageStyleAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("PageStyleAction: %w", err)
	}

	switch name {
	case "ListPending":
		a.ListPending = unitSet()
	case "Approve":
		a.Approve = &PageImageDecision{}
		return decodeVariant("PageStyleAction", name, payload, a.Approve)
	case "Reject":
		a.Reject = &PageImageDecision{}
		return decodeVariant("PageStyleAction", name, payload, a.Reject)
	default:
		return errUnknownVariant("PageStyleAction", name)
	}

	return expectUnit("PageStyleAction", name, payload)
}

func (a PageStyleAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListPending != nil:
		return encodeUnit("ListPending")
	case a.Approve != nil:
		return encodeVariant("Approve", a.Approve)
	case a.Reject != nil:
		return encodeVariant("Reject", a.Reject)
	default:
		return nil, fmt.Errorf("PageStyleAction: no variant set")
	}
}
