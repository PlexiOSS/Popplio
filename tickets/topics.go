// Copyright (C) 2026 NodeByte LTD

package tickets

import "popplio/types"

var Topics = []types.TicketTopic{
	{ID: "billing", Name: "Billing & Payments"},
	{ID: "account", Name: "Account Issues"},
	{ID: "bug_report", Name: "Bug Report"},
	{ID: "abuse_report", Name: "Report Abuse"},
	{ID: "other", Name: "Other"},
}

func FindTopic(id string) *types.TicketTopic {
	for _, t := range Topics {
		if t.ID == id {
			return &t
		}
	}
	return nil
}
