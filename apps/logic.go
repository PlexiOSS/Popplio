package apps

import (
	"context"
	"errors"
	"fmt"
	"popplio/perms"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// certBotOwnerIDs returns whoever should be treated as a bot's owner for
// cert-role purposes: the direct owner if it's not team-owned, or every
// member of its owning team otherwise — a bot is always one or the other,
// never both (see the owner_team_oneof check constraint).
func certBotOwnerIDs(ctx context.Context, botID string) ([]string, error) {
	var owner pgtype.Text
	var teamOwner pgtype.UUID

	err := state.Pool.QueryRow(ctx, "SELECT owner, team_owner FROM bots WHERE bot_id = $1", botID).Scan(&owner, &teamOwner)

	if err != nil {
		return nil, fmt.Errorf("error fetching bot owner info: %w", err)
	}

	if owner.Valid && owner.String != "" {
		return []string{owner.String}, nil
	}

	if !teamOwner.Valid {
		return nil, nil
	}

	return teamMemberIDs(ctx, teamOwner)
}

// certServerOwnerIDs returns every member of a server's owning team —
// servers are always team-owned, there's no direct-owner equivalent.
func certServerOwnerIDs(ctx context.Context, serverID string) ([]string, error) {
	var teamOwner pgtype.UUID

	err := state.Pool.QueryRow(ctx, "SELECT team_owner FROM servers WHERE server_id = $1", serverID).Scan(&teamOwner)

	if err != nil {
		return nil, fmt.Errorf("error fetching server owner info: %w", err)
	}

	if !teamOwner.Valid {
		return nil, nil
	}

	return teamMemberIDs(ctx, teamOwner)
}

func teamMemberIDs(ctx context.Context, teamID pgtype.UUID) ([]string, error) {
	rows, err := state.Pool.Query(ctx, "SELECT user_id FROM team_members WHERE team_id = $1", teamID)

	if err != nil {
		return nil, fmt.Errorf("error fetching team members: %w", err)
	}

	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning team member id: %w", err)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// grantCertOwnerRoles gives BotDeveloper/CertifiedDeveloper to every owner
// ID who's actually a member of the main guild — used for both certified
// bots and certified servers, since there's no separate server-owner role
// convention. Best-effort throughout: an owner not being in the guild, or a
// Discord API error, is logged rather than failing the caller, since role
// grants are a side effect of certification, not a precondition for it.
func grantCertOwnerRoles(ownerIDs []string) {
	for _, ownerID := range ownerIDs {
		uid, err := snowflake.Parse(ownerID)

		if err != nil {
			state.Logger.Warn("Could not parse owner ID for cert role grant", zap.String("owner_id", ownerID), zap.Error(err))
			continue
		}

		if _, ok := state.Discord.Caches().Member(state.Config.Servers.Main, uid); !ok {
			continue
		}

		if err := state.Discord.Rest().AddMemberRole(state.Config.Servers.Main, uid, state.Config.Roles.BotDeveloper, rest.WithReason("Owner of a certified listing")); err != nil {
			state.Logger.Warn("Failed to grant BotDeveloper role to cert owner", zap.String("owner_id", ownerID), zap.Error(err))
		}

		if err := state.Discord.Rest().AddMemberRole(state.Config.Servers.Main, uid, state.Config.Roles.CertifiedDeveloper, rest.WithReason("Owner of a certified listing")); err != nil {
			state.Logger.Warn("Failed to grant CertifiedDeveloper role to cert owner", zap.String("owner_id", ownerID), zap.Error(err))
		}
	}
}

var permBotResubmit = perms.EntityResubmitBots
var permBotCertify = perms.EntityCertifyBots
var permServerCertify = perms.EntityCertifyServers

var ErrNoPersist = errors.New("no persist") // This error should be returned when the app should not be persisted to the database for review

// Certification used to require servers >= 100 AND unique clicks >= 30 —
// both, no exceptions. That shut out bots/servers that were genuinely
// strong on one metric (e.g. very high votes, modest server count) just
// because they hadn't cleared every bar. It's now an OR across whichever
// reach/engagement stat the listing actually has, each bar lowered to
// match, plus a (new, but lenient) minimum listed-age floor so a
// same-day submission can't certify off a launch-day traffic spike.
const (
	certMinBotServers    = 50
	certMinServerMembers = 100
	certMinUniqueClicks  = 15
	certMinVotes         = 50
	certMinListedAge     = 3 * 24 * time.Hour
)

// checkCertStats applies the shared OR-of-three-metrics rule. reachLabel
// names whatever reachMin is measured in ("servers" for bots, "members"
// for servers) for the error message.
func checkCertStats(reach, uniqueClicks, votes int64, reachLabel string, reachMin int64) error {
	if reach >= reachMin || uniqueClicks >= certMinUniqueClicks || votes >= certMinVotes {
		return nil
	}

	return fmt.Errorf(
		"not enough reach yet to certify — needs at least %d %s (has %d), %d unique clicks (has %d), or %d votes (has %d)",
		reachMin, reachLabel, reach, certMinUniqueClicks, uniqueClicks, certMinVotes, votes,
	)
}

func extraLogicResubmit(d uapi.RouteData, p types.Position, answers map[string]string) error {
	// Get the bot ID
	botID, ok := answers["id"]

	if !ok {
		return errors.New("bot ID not found")
	}

	// Get the bot
	var botType string
	err := state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

	if err != nil {
		return fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
	}

	entityPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", botID)

	if err != nil {
		return fmt.Errorf("error getting user bot perms: %w", err)
	}

	// Check if user has TeamPermissionResubmitBots
	if !entityPerms.Has(permBotResubmit) {
		return errors.New("you do not have permission to resubmit bots")
	}

	if botType == "approved" || botType == "pending" || botType == "certified" {
		return errors.New("bot is approved, pending or certified | state=" + botType)
	}

	// Set the bot type to pending
	_, err = state.Pool.Exec(d.Context, "UPDATE bots SET type = 'pending', claimed_by = NULL WHERE bot_id = $1", botID)

	if err != nil {
		return fmt.Errorf("error setting bot type to pending: %w", err)
	}

	user, err := dovewing.GetUser(d.Context, botID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error getting discord user: %w", err)
	}

	// Send an embed to the bot logs channel
	_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
		Content: state.Config.Meta.UrgentMentions,
		Embeds: []discord.Embed{
			{
				Title:       "Bot Resubmitted!",
				URL:         state.Config.Sites.Frontend + "/bots/" + botID,
				Description: "User <@" + d.Auth.ID + "> has resubmitted their bot",
				Color:       0x00ff00,
				Fields: []discord.EmbedField{
					{
						Name:  "Bot ID",
						Value: botID,
					},
					{
						Name:  "Bot Name",
						Value: user.DisplayName + " (" + user.ID + ")",
					},
					{
						Name:   "Reason",
						Value:  answers["reason"],
						Inline: validators.TruePtr,
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("error sending embed to bot logs channel: %w", err)
	}

	return nil // Should it be ErrNoPersist?
}

func extraLogicCert(d uapi.RouteData, p types.Position, answers map[string]string) error {
	// Get the bot ID
	botID, ok := answers["id"]

	if !ok {
		return errors.New("bot ID not found")
	}

	// Get the bot
	var botType string
	var serverCount, uniqueClicks, approxVotes int64
	var createdAt time.Time
	err := state.Pool.QueryRow(d.Context, "SELECT type, servers, cardinality(unique_clicks), approximate_votes, created_at FROM bots WHERE bot_id = $1", botID).Scan(&botType, &serverCount, &uniqueClicks, &approxVotes, &createdAt)

	if err != nil {
		return fmt.Errorf("error getting bot info, does the bot exist?: %w", err)
	}

	entityPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "bot", botID)

	if err != nil {
		return fmt.Errorf("error getting user bot perms: %w", err)
	}

	// Check if user has TeamPermissionCertifyBots
	if !entityPerms.Has(permBotCertify) {
		return errors.New("you do not have permission to certify bots")
	}

	if botType != "approved" {
		return errors.New("bot is not approved | state=" + botType)
	}

	if time.Since(createdAt) < certMinListedAge {
		return fmt.Errorf("bot must be listed for at least %s before it can be certified", certMinListedAge)
	}

	return checkCertStats(serverCount, uniqueClicks, approxVotes, "servers", certMinBotServers)
}

func extraLogicCertServer(d uapi.RouteData, p types.Position, answers map[string]string) error {
	// Get the server ID
	serverID, ok := answers["id"]

	if !ok {
		return errors.New("server ID not found")
	}

	// Get the server
	var serverType string
	var memberCount, uniqueClicks, approxVotes int64
	var createdAt time.Time
	err := state.Pool.QueryRow(d.Context, "SELECT type, total_members, cardinality(unique_clicks), approximate_votes, created_at FROM servers WHERE server_id = $1", serverID).Scan(&serverType, &memberCount, &uniqueClicks, &approxVotes, &createdAt)

	if err != nil {
		return fmt.Errorf("error getting server info, does the server exist?: %w", err)
	}

	entityPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "server", serverID)

	if err != nil {
		return fmt.Errorf("error getting user server perms: %w", err)
	}

	if !entityPerms.Has(permServerCertify) {
		return errors.New("you do not have permission to certify servers")
	}

	if serverType != "approved" {
		return errors.New("server is not approved | state=" + serverType)
	}

	if time.Since(createdAt) < certMinListedAge {
		return fmt.Errorf("server must be listed for at least %s before it can be certified", certMinListedAge)
	}

	return checkCertStats(memberCount, uniqueClicks, approxVotes, "members", certMinServerMembers)
}

func reviewLogicBanAppeal(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	if approve {
		// Unban user

		if len(reason) > 384 {
			return errors.New("reason must be less than 384 characters")
		}

		userId, err := snowflake.Parse(resp.UserID)

		if err != nil {
			return fmt.Errorf("error parsing user ID: %w", err)
		}

		err = state.Discord.Rest().DeleteBan(
			state.Config.Servers.Main,
			userId,
			rest.WithReason("Ban appeal accepted by "+d.Auth.ID+" | "+reason),
		)

		if err != nil {
			return err
		}
	} else {
		// Denial is always possible
		return nil
	}

	return nil
}

func reviewLogicCert(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	if approve {
		// Get the bot ID
		botID, ok := resp.Answers["id"]

		if !ok {
			return errors.New("bot ID not found")
		}

		botIdSnow, err := snowflake.Parse(botID)

		if err != nil {
			return fmt.Errorf("error parsing bot ID: %w", err)
		}

		// Get the bot
		var botType string
		err = state.Pool.QueryRow(d.Context, "SELECT type FROM bots WHERE bot_id = $1", botID).Scan(&botType)

		if err != nil {
			return fmt.Errorf("error getting bot type, does the bot exist?: %w", err)
		}

		if botType == "certified" {
			return nil // Just approve the review
		}

		if botType != "approved" {
			return errors.New("bot is not approved | state=" + botType + ". Please deny the certification until approved")
		}

		// Set the bot type to certified
		_, err = state.Pool.Exec(d.Context, "UPDATE bots SET type = 'certified' WHERE bot_id = $1", botID)

		if err != nil {
			return fmt.Errorf("error setting bot type to certified: %w", err)
		}

		// Give the bot its own role. Best-effort: the bot might not be a
		// member of the main guild, which shouldn't block certification
		// from actually persisting — this used to return an error here,
		// which meant the caller (manage_app) never ran the UPDATE apps
		// SET state = 'approved' below it, leaving the app stuck "pending"
		// forever even though the bot was already certified in the bots
		// table.
		if err := state.Discord.Rest().AddMemberRole(state.Config.Servers.Main, botIdSnow, state.Config.Roles.CertBot); err != nil {
			state.Logger.Warn("Failed to give certified bot role to bot", zap.String("bot_id", botID), zap.Error(err))
		}

		// Give the owner(s) their developer roles too — this used to be a
		// fully manual step (ibb!getbotroles), so an owner who never ran
		// the command never got anything.
		if ownerIDs, err := certBotOwnerIDs(d.Context, botID); err != nil {
			state.Logger.Warn("Failed to resolve bot owners for cert role grant", zap.String("bot_id", botID), zap.Error(err))
		} else {
			grantCertOwnerRoles(ownerIDs)
		}

		// Send an embed to the bot logs channel
		_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Bot Certified!",
					URL:         state.Config.Sites.Frontend + "/bots/" + botID,
					Description: "<@" + d.Auth.ID + "> has certified bot <@" + botID + ">",
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Bot ID",
							Value: botID,
						},
						{
							Name:  "Reason",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "Owner roles are applied automatically if you're in the server — run ibb!getbotroles anytime to re-sync",
					},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("error sending embed to bot logs channel, but successfully certified bot: %w", err)
		}
	} else {
		// Denial is always possible
		return nil
	}

	return nil
}

func reviewLogicCertServer(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	if approve {
		// Get the server ID
		serverID, ok := resp.Answers["id"]

		if !ok {
			return errors.New("server ID not found")
		}

		// Get the server
		var serverType string
		err := state.Pool.QueryRow(d.Context, "SELECT type FROM servers WHERE server_id = $1", serverID).Scan(&serverType)

		if err != nil {
			return fmt.Errorf("error getting server type, does the server exist?: %w", err)
		}

		if serverType == "certified" {
			return nil // Just approve the review
		}

		if serverType != "approved" {
			return errors.New("server is not approved | state=" + serverType + ". Please deny the certification until approved")
		}

		// Set the server type to certified
		_, err = state.Pool.Exec(d.Context, "UPDATE servers SET type = 'certified' WHERE server_id = $1", serverID)

		if err != nil {
			return fmt.Errorf("error setting server type to certified: %w", err)
		}

		// A listed server isn't itself a member of the main guild the way a
		// bot is, so there's no server-equivalent of the CertBot role to
		// grant here — but its owner(s) still get the same developer roles
		// a certified bot's owner would, reusing that convention rather
		// than inventing a separate server-owner role.
		if ownerIDs, err := certServerOwnerIDs(d.Context, serverID); err != nil {
			state.Logger.Warn("Failed to resolve server owners for cert role grant", zap.String("server_id", serverID), zap.Error(err))
		} else {
			grantCertOwnerRoles(ownerIDs)
		}

		// Send an embed to the bot logs channel
		_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.BotLogs, discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Server Certified!",
					URL:         state.Config.Sites.Frontend + "/servers/" + serverID,
					Description: "<@" + d.Auth.ID + "> has certified server " + serverID,
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Server ID",
							Value: serverID,
						},
						{
							Name:  "Reason",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "Owner roles are applied automatically to team members who are in the server",
					},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("error sending embed to bot logs channel, but successfully certified server: %w", err)
		}
	} else {
		// Denial is always possible
		return nil
	}

	return nil
}

func reviewLogicStaff(d uapi.RouteData, resp types.AppResponse, reason string, approve bool) error {
	userId, err := snowflake.Parse(resp.UserID)

	if err != nil {
		return fmt.Errorf("error parsing user ID: %w", err)
	}

	if approve {
		err := state.Discord.Rest().AddMemberRole(state.Config.Servers.Main, userId, state.Config.Roles.AwaitingStaff)

		if err != nil {
			return err
		}

		// DM the user
		dmchan, err := state.Discord.Rest().CreateDMChannel(userId)

		if err != nil {
			return errors.New("could not send DM, please ask the user to accept DMs from server members")
		}

		if len(reason) > 1024 {
			return errors.New("reason must be 1024 characters or less")
		}

		_, err = state.Discord.Rest().CreateMessage(dmchan.ID(), discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Staff Application Whitelisted",
					Description: "Your staff application has been whitelisted for onboarding! Please ping any manager at #staff-only in our discord server to get started.",
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Feedback",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "Congratulations!",
					},
				},
			},
		})

		if err != nil {
			return errors.New("could not send DM, please ask the user to accept DMs from server members")
		}

		return nil
	} else {
		if strings.HasPrefix(reason, "MANUALLYNOTIFIED ") {
			state.Logger.Info("forcing denial of staff application that was manually notified by a manager", zap.String("userID", resp.UserID))
			return nil
		}

		// Attempt to DM the user on denial
		dmchan, err := state.Discord.Rest().CreateDMChannel(userId)

		if err != nil {
			return fmt.Errorf("could not create DM channel with user, please inform them manually, then deny with reason of 'MANUALLYNOTIFIED <your reason here>': %w", err)
		}

		_, err = state.Discord.Rest().CreateMessage(dmchan.ID(), discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					Title:       "Staff Application Denied",
					Description: "Unfortunately, we have denied your staff application for Omniplex. You may reapply later if you wish to",
					Color:       0x00ff00,
					Fields: []discord.EmbedField{
						{
							Name:  "Reason",
							Value: reason,
						},
					},
					Footer: &discord.EmbedFooter{
						Text: "Better luck next time?",
					},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("could not send DM, please inform them manually, then deny with reason of 'MANUALLYNOTIFIED <your reason here>': %w", err)
		}

		return nil
	}
}
