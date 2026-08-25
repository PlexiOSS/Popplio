package impls

import (
	"errors"
	"time"

	"popplio/arcadia/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
)

// Colours used by the mod-log embeds. BLURPLE is serenity's Colour::BLURPLE,
// which the Claim embed uses.
const (
	ColourBlurple = 0x7289DA
	ColourGreen   = 0x00ff00
	ColourRed     = 0xFF0000
	// ColourRedLower is the lowercase 0xff0000 literal the certify embeds use.
	// Identical value, kept separate only so the port reads like the source.
	ColourRedLower = 0xff0000
)

var (
	truePtr  = boolPtr(true)
	falsePtr = boolPtr(false)
)

func boolPtr(b bool) *bool { return &b }

// InlineTrue and InlineFalse are the embed field inline flags.
func InlineTrue() *bool  { return truePtr }
func InlineFalse() *bool { return falsePtr }

// MemberOnGuild reports whether a user is a cached member of a guild. A guild
// that is not cached reads as "not a member", as upstream does.
func MemberOnGuild(guildID snowflake.ID, userID snowflake.ID) bool {
	_, ok := dclient.Get().Caches().Member(guildID, userID)
	return ok
}

// SendModLog posts a message to the configured mod-logs channel.
func SendModLog(msg discord.MessageCreate) error {
	_, err := dclient.Get().Rest().CreateMessage(state.Config.Channels.ModLogs, msg)
	return err
}

// SendChannel posts a message to an arbitrary channel.
//
// A zero id means the channel was never configured. Discord answers that with
// "10003: Unknown Channel", which sends whoever reads the error hunting for a
// deleted channel instead of a missing config key, so it is caught here.
func SendChannel(channelID snowflake.ID, msg discord.MessageCreate) error {
	if channelID == 0 {
		return errors.New("channel is not configured (check the `channels` section of config.yaml)")
	}

	_, err := dclient.Get().Rest().CreateMessage(channelID, msg)

	return err
}

// AddRole grants a role, recording an audit-log reason.
func AddRole(guildID, userID, roleID snowflake.ID, reason string) error {
	return dclient.Get().Rest().AddMemberRole(guildID, userID, roleID, rest.WithReason(reason))
}

// RemoveRole revokes a role, recording an audit-log reason.
func RemoveRole(guildID, userID, roleID snowflake.ID, reason string) error {
	return dclient.Get().Rest().RemoveMemberRole(guildID, userID, roleID, rest.WithReason(reason))
}

// KickMember removes a member from a guild, recording an audit-log reason.
func KickMember(guildID, userID snowflake.ID, reason string) error {
	return dclient.Get().Rest().RemoveMember(guildID, userID, rest.WithReason(reason))
}

// BanMember bans a member from a guild, recording an audit-log reason.
// deleteMessages controls how much of their recent message history is purged
// along with the ban (0 deletes nothing).
func BanMember(guildID, userID snowflake.ID, deleteMessages time.Duration, reason string) error {
	return dclient.Get().Rest().AddBan(guildID, userID, deleteMessages, rest.WithReason(reason))
}

// TimeoutMember puts a member in timeout until the given time, recording an
// audit-log reason. Discord caps this at 28 days from now; callers are
// expected to have already validated that.
func TimeoutMember(guildID, userID snowflake.ID, until time.Time, reason string) error {
	_, err := dclient.Get().Rest().UpdateMember(guildID, userID, discord.MemberUpdate{
		CommunicationDisabledUntil: json.NewNullablePtr(until),
	}, rest.WithReason(reason))

	return err
}

// PurgeMessages bulk-deletes messages in a channel, recording an audit-log
// reason. Discord's bulk-delete endpoint only accepts messages younger than
// 14 days and refuses fewer than 2 ids; callers are expected to have already
// filtered for both.
func PurgeMessages(channelID snowflake.ID, messageIDs []snowflake.ID, reason string) error {
	return dclient.Get().Rest().BulkDeleteMessages(channelID, messageIDs, rest.WithReason(reason))
}

// GetRecentMessages fetches up to limit of the most recent messages in a
// channel.
func GetRecentMessages(channelID snowflake.ID, limit int) ([]discord.Message, error) {
	return dclient.Get().Rest().GetMessages(channelID, 0, 0, 0, limit)
}

// DeleteMessage removes a single message, recording an audit-log reason.
func DeleteMessage(channelID, messageID snowflake.ID, reason string) error {
	return dclient.Get().Rest().DeleteMessage(channelID, messageID, rest.WithReason(reason))
}

// LockChannel denies @everyone's Send Messages permission in a channel,
// preserving every other bit already on the overwrite.
func LockChannel(guildID, channelID snowflake.ID, reason string) error {
	return setEveryoneSendMessages(guildID, channelID, false, reason)
}

// UnlockChannel restores @everyone's ability to send messages in a channel by
// clearing the Send Messages deny bit only -- it does not touch any other
// overwrite the channel already had.
func UnlockChannel(guildID, channelID snowflake.ID, reason string) error {
	return setEveryoneSendMessages(guildID, channelID, true, reason)
}

// setEveryoneSendMessages flips the Send Messages bit on the @everyone
// permission overwrite for a channel. The @everyone role's id is always the
// guild's own id, which is what makes this a lookup by guildID rather than a
// separate role-resolution step.
func setEveryoneSendMessages(guildID, channelID snowflake.ID, allow bool, reason string) error {
	existing, err := dclient.Get().Rest().GetPermissionOverwrite(channelID, guildID)

	var curAllow, curDeny discord.Permissions

	// A missing overwrite (err != nil, or a nil/unexpected-typed result)
	// starts from a zero-value one and lets the update below create it.
	if err == nil && existing != nil {
		if role, ok := (*existing).(discord.RolePermissionOverwrite); ok {
			curAllow, curDeny = role.Allow, role.Deny
		}
	}

	if allow {
		curDeny &^= discord.PermissionSendMessages
	} else {
		curDeny |= discord.PermissionSendMessages
		curAllow &^= discord.PermissionSendMessages
	}

	return dclient.Get().Rest().UpdatePermissionOverwrite(channelID, guildID,
		discord.RolePermissionOverwriteUpdate{Allow: &curAllow, Deny: &curDeny}, rest.WithReason(reason))
}

// SendDM opens (or reuses) a DM channel with a user and sends a message.
// Best-effort by nature — a user with DMs closed to the bot makes this fail,
// which callers that also log the action elsewhere should treat as
// non-fatal.
func SendDM(userID snowflake.ID, msg discord.MessageCreate) error {
	channel, err := dclient.Get().Rest().CreateDMChannel(userID)

	if err != nil {
		return err
	}

	_, err = dclient.Get().Rest().CreateMessage(channel.ID(), msg)

	return err
}

// Footer builds an embed footer.
func Footer(text string) *discord.EmbedFooter {
	return &discord.EmbedFooter{Text: text}
}
