package types

import (
	"encoding/json"
	"fmt"
)

type PlatformUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	DisplayName string `json:"display_name"`
	Bot         bool   `json:"bot"`
	Status      string `json:"status"`
}

func (p PlatformUser) Equals(other PlatformUser) bool {
	return p.ID == other.ID
}

type PartialBot struct {
	BotID                string       `json:"bot_id"`
	User                 PlatformUser `json:"user"`
	Short                string       `json:"short"`
	Type                 string       `json:"type"`
	Votes                int32        `json:"votes"`
	Shards               int32        `json:"shards"`
	Library              string       `json:"library"`
	InviteClicks         int32        `json:"invite_clicks"`
	Clicks               int32        `json:"clicks"`
	Servers              int32        `json:"servers"`
	ClaimedBy            *string      `json:"claimed_by"`
	LastClaimed          *Timestamp   `json:"last_claimed"`
	ApprovalNote         string       `json:"approval_note"`
	Mentionable          []string     `json:"mentionable"`
	Invite               string       `json:"invite"`
	ClientID             string       `json:"client_id"`
	ModerationFlagged    bool         `json:"moderation_flagged"`
	ModerationCategories []string     `json:"moderation_categories"`
}

type PartialServer struct {
	ServerID             string     `json:"server_id"`
	Name                 string     `json:"name"`
	Avatar               string     `json:"avatar"`
	TotalMembers         int32      `json:"total_members"`
	OnlineMembers        int32      `json:"online_members"`
	Short                string     `json:"short"`
	Type                 string     `json:"type"`
	Votes                int32      `json:"votes"`
	InviteClicks         int32      `json:"invite_clicks"`
	Clicks               int32      `json:"clicks"`
	NSFW                 bool       `json:"nsfw"`
	DiscordNSFWLevel     int32      `json:"discord_nsfw_level"`
	NSFWChannelCount     int32      `json:"nsfw_channel_count"`
	Tags                 []string   `json:"tags"`
	Premium              bool       `json:"premium"`
	ClaimedBy            *string    `json:"claimed_by"`
	LastClaimed          *Timestamp `json:"last_claimed"`
	ApprovalNote         string     `json:"approval_note"`
	Mentionable          []string   `json:"mentionable"`
	ModerationFlagged    bool       `json:"moderation_flagged"`
	ModerationCategories []string   `json:"moderation_categories"`
}

type PartialPack struct {
	URL        string       `json:"url"`
	Name       string       `json:"name"`
	Short      string       `json:"short"`
	PackType   string       `json:"pack_type"`
	Owner      PlatformUser `json:"owner"`
	Votes      int32        `json:"votes"`
	Tags       []string     `json:"tags"`
	VoteBanned bool         `json:"vote_banned"`
}

type PartialTeam struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Short      string   `json:"short"`
	Votes      int32    `json:"votes"`
	Tags       []string `json:"tags"`
	NSFW       bool     `json:"nsfw"`
	VoteBanned bool     `json:"vote_banned"`
}

type PartialUser struct {
	User       PlatformUser `json:"user"`
	Staff      bool         `json:"staff"`
	Banned     bool         `json:"banned"`
	VoteBanned bool         `json:"vote_banned"`
}

type PartialEntity struct {
	Bot    *PartialBot
	Server *PartialServer
	Pack   *PartialPack
	Team   *PartialTeam
	User   *PartialUser
}

func (e *PartialEntity) UnmarshalJSON(data []byte) error {
	*e = PartialEntity{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("PartialEntity: %w", err)
	}

	var into any

	switch name {
	case "Bot":
		e.Bot = &PartialBot{}
		into = e.Bot
	case "Server":
		e.Server = &PartialServer{}
		into = e.Server
	case "Pack":
		e.Pack = &PartialPack{}
		into = e.Pack
	case "Team":
		e.Team = &PartialTeam{}
		into = e.Team
	case "User":
		e.User = &PartialUser{}
		into = e.User
	default:
		return errUnknownVariant("PartialEntity", name)
	}

	return decodeVariant("PartialEntity", name, payload, into)
}

func (e PartialEntity) MarshalJSON() ([]byte, error) {
	switch {
	case e.Bot != nil:
		return encodeVariant("Bot", e.Bot)
	case e.Server != nil:
		return encodeVariant("Server", e.Server)
	case e.Pack != nil:
		return encodeVariant("Pack", e.Pack)
	case e.Team != nil:
		return encodeVariant("Team", e.Team)
	case e.User != nil:
		return encodeVariant("User", e.User)
	default:
		return nil, fmt.Errorf("PartialEntity: no variant set")
	}
}

type RPCLogEntry struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Method    string          `json:"method"`
	State     string          `json:"state"`
	Data      json.RawMessage `json:"data"`
	CreatedAt Timestamp       `json:"created_at"`
}

type BaseAnalytics struct {
	BotCounts       map[string]int64 `json:"bot_counts"`
	ServerCounts    map[string]int64 `json:"server_counts"`
	TicketCounts    map[string]int64 `json:"ticket_counts"`
	TotalUsers      int64            `json:"total_users"`
	ChangelogsCount int64            `json:"changelogs_count"`
}
