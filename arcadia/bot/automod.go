package bot

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// inviteRe matches discord.gg and discord(app).com/invite links. There is no
// per-guild vanity whitelist — this codebase has no per-guild settings
// concept at all (config.yaml, loaded once at startup, is the only settings
// store) — so every invite link is treated as a violation for now. If that
// proves too aggressive, the fix is a static Arcadia.AutoModInviteWhitelist
// []string config field, the same shape ProtectedBots already is, not a DB
// table.
var inviteRe = regexp.MustCompile(`(?i)(discord\.gg|discord(?:app)?\.com/invite)/[a-z0-9-]+`)

const (
	spamWindow       = 7 * time.Second
	spamThreshold    = 5 // messages within spamWindow from one user in one channel
	massMentionLimit = 5 // distinct user/role mentions in one message
)

// spamSeen is a tiny in-memory sliding window per (guild, channel, user).
// Intentionally not persisted: a restart resetting everyone's spam counter
// is an acceptable cold-start cost for a heuristic this cheap, and keeping
// it out of Postgres avoids a write on every single message.
var (
	spamMu   sync.Mutex
	spamSeen = map[string][]time.Time{}
)

func spamKey(guildID, channelID, userID snowflake.ID) string {
	return guildID.String() + ":" + channelID.String() + ":" + userID.String()
}

// recordAndCheckSpam appends the current message time and reports whether
// the user has crossed spamThreshold within spamWindow.
func recordAndCheckSpam(guildID, channelID, userID snowflake.ID, at time.Time) bool {
	spamMu.Lock()
	defer spamMu.Unlock()

	key := spamKey(guildID, channelID, userID)
	cutoff := at.Add(-spamWindow)

	kept := spamSeen[key][:0]

	for _, t := range spamSeen[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	kept = append(kept, at)
	spamSeen[key] = kept

	return len(kept) >= spamThreshold
}

// scanMessage passively inspects one message for spam, invite links, and
// mass mentions in the main/testing servers only, taking action and logging
// on a match. Never touches the staff server.
func scanMessage(ctx context.Context, e *events.MessageCreate) {
	if e.GuildID == nil {
		return
	}

	guildID := *e.GuildID

	if guildID != state.Config.Servers.Main && guildID != state.Config.Servers.Testing {
		return
	}

	msg := e.Message
	var reason string

	switch {
	case inviteRe.MatchString(msg.Content):
		reason = "posted an invite link"
	case len(msg.Mentions)+len(msg.MentionRoles) >= massMentionLimit:
		reason = fmt.Sprintf("mass-mentioned %d users/roles in one message", len(msg.Mentions)+len(msg.MentionRoles))
	case recordAndCheckSpam(guildID, e.ChannelID, msg.Author.ID, time.Now()):
		reason = fmt.Sprintf("sent %d+ messages within %s", spamThreshold, spamWindow)
	default:
		return
	}

	takeAutoModAction(ctx, guildID, e.ChannelID, msg, reason)
}

// takeAutoModAction deletes the offending message, DMs a warning, records a
// mod_cases row, and posts to the mod-logs channel — all attributed to the
// bot itself, never to a staff member, since no staff member acted.
func takeAutoModAction(ctx context.Context, guildID, channelID snowflake.ID, msg discord.Message, reason string) {
	botID := dclient.Get().ApplicationID()

	if err := dclient.Get().Rest().DeleteMessage(channelID, msg.ID); err != nil {
		state.Logger.Error("Automod failed to delete message", zap.Error(err))
	}

	_ = impls.SendDM(msg.Author.ID, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       "Automated warning",
			Description: fmt.Sprintf("Your message in <#%s> was removed: %s.", channelID, reason),
			Color:       impls.ColourRed,
		}},
	})

	if err := writeModCase(ctx, modCase{
		GuildID: guildID, UserID: msg.Author.ID, ModeratorID: botID,
		Action: "automod", Reason: reason,
	}); err != nil {
		state.Logger.Error("Failed to record automod mod case", zap.Error(err))
	}

	err := impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title: "Auto-mod Action",
			Fields: []discord.EmbedField{
				{Name: "User", Value: fmt.Sprintf("<@%s> (%s)", msg.Author.ID, msg.Author.ID), Inline: impls.InlineTrue()},
				{Name: "Staff", Value: "Automated (Arcadia)", Inline: impls.InlineTrue()},
				{Name: "Reason", Value: reason, Inline: impls.InlineFalse()},
				{Name: "Channel", Value: fmt.Sprintf("<#%s>", channelID), Inline: impls.InlineTrue()},
			},
			Color: impls.ColourRed,
		}},
	})

	if err != nil {
		state.Logger.Error("Failed to post automod mod-log", zap.Error(err))
	}
}
