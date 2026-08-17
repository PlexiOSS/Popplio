// Package tickets implements Omniplex's standalone support ticket system.
//
// Tickets are plain web tickets stored in the existing `tickets` table —
// there is no Discord channel behind them (channel_id is left empty), no
// message encryption (enc_key is left null), and message IDs are
// synthesized Discord-format snowflakes (via disgoorg/snowflake's New)
// purely so GET /tickets/{id}'s existing snowflake.Parse-based timestamp
// decoding keeps working unchanged for both old Discord-linked tickets and
// new standalone ones.
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
