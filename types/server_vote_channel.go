// Copyright (C) 2026 NodeByte LTD

package types

type ServerVoteChannelOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ServerVoteChannel struct {
	ChannelID *string                   `json:"channel_id" description:"The channel vote messages are posted in, or null if they're off"`
	Channels  []ServerVoteChannelOption `json:"channels" description:"The server's text and announcement channels"`
	Available bool                      `json:"available" description:"Whether the Infernoplex bot can see the server. When false, channels is empty"`
}

type ServerVoteChannelUpdate struct {
	ChannelID *string `json:"channel_id" validate:"omitempty,numeric" msg:"channel_id must be a channel ID, or null to turn vote messages off"`
}
