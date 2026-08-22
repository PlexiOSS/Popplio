package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// The review queue as the panel sees it, and the user lookup beside it.
//
// partialBots is the batched manager resolution: upstream resolved each bot's
// managers with two queries inside the loop, so an N-bot response cost 2N round
// trips. It is three queries total here, with the same JSON out.

func (s *Server) getUser(ctx context.Context, q *types.QGetUser) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	user, err := impls.GetPlatformUser(ctx, q.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, user), nil
}

type botQueueRow struct {
	BotID                string     `db:"bot_id"`
	ClientID             string     `db:"client_id"`
	LastClaimed          *time.Time `db:"last_claimed"`
	ClaimedBy            *string    `db:"claimed_by"`
	Type                 string     `db:"type"`
	ApprovalNote         string     `db:"approval_note"`
	Short                string     `db:"short"`
	Invite               string     `db:"invite"`
	ApproximateVotes     int32      `db:"approximate_votes"`
	Shards               int32      `db:"shards"`
	Library              string     `db:"library"`
	InviteClicks         int32      `db:"invite_clicks"`
	Clicks               int32      `db:"clicks"`
	Servers              int32      `db:"servers"`
	ModerationFlagged    bool       `db:"moderation_flagged"`
	ModerationCategories []string   `db:"moderation_categories"`
}

func (s *Server) botQueue(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	// Public to all staff: no permission check.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	rows, err := state.Pool.Query(ctx,
		`SELECT bot_id, client_id, last_claimed, claimed_by, type, approval_note, short,
                invite, approximate_votes, shards, library, invite_clicks, clicks, servers,
                moderation_flagged, moderation_categories
                FROM bots WHERE type = 'pending' OR type = 'claimed' ORDER BY created_at`)

	if err != nil {
		return response{}, newError(err)
	}

	queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[botQueueRow])

	if err != nil {
		return response{}, newError(err)
	}

	return s.partialBots(ctx, queue)
}

func (s *Server) serverQueue(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	// Public to all staff: no permission check, matching botQueue.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	rows, err := state.Pool.Query(ctx,
		`SELECT server_id, name, avatar, total_members, online_members, short, type, approval_note,
                approximate_votes, invite_clicks, clicks, nsfw, discord_nsfw_level, nsfw_channel_count,
                tags, premium, claimed_by, last_claimed, moderation_flagged, moderation_categories
                FROM servers WHERE type = 'pending' OR type = 'claimed' ORDER BY created_at`)

	if err != nil {
		return response{}, newError(err)
	}

	queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[searchServerRow])

	if err != nil {
		return response{}, newError(err)
	}

	return s.partialServers(ctx, queue)
}

// partialServers renders a server list, batching manager resolution the same
// way partialBots does.
func (s *Server) partialServers(ctx context.Context, queue []searchServerRow) (response, error) {
	serverIDs := make([]string, 0, len(queue))

	for _, server := range queue {
		serverIDs = append(serverIDs, server.ServerID)
	}

	managers, err := impls.GetServerManagers(ctx, serverIDs)

	if err != nil {
		return response{}, newError(err)
	}

	servers := make([]types.PartialEntity, 0, len(queue))

	for _, server := range queue {
		servers = append(servers, types.PartialEntity{Server: &types.PartialServer{
			ServerID:             server.ServerID,
			Name:                 server.Name,
			Avatar:               server.Avatar,
			TotalMembers:         server.TotalMembers,
			OnlineMembers:        server.OnlineMembers,
			Short:                server.Short,
			Type:                 server.Type,
			ApprovalNote:         server.ApprovalNote,
			Votes:                server.ApproximateVotes,
			InviteClicks:         server.InviteClicks,
			Clicks:               server.Clicks,
			NSFW:                 server.NSFW,
			DiscordNSFWLevel:     server.DiscordNSFWLevel,
			NSFWChannelCount:     server.NSFWChannelCount,
			Tags:                 types.NonNilStrings(server.Tags),
			Premium:              server.Premium,
			ClaimedBy:            server.ClaimedBy,
			LastClaimed:          types.TimestampPtr(server.LastClaimed),
			Mentionable:          managers[server.ServerID].Mentionables(),
			ModerationFlagged:    server.ModerationFlagged,
			ModerationCategories: types.NonNilStrings(server.ModerationCategories),
		}})
	}

	return writeJSON(http.StatusOK, servers), nil
}

func (s *Server) partialBots(ctx context.Context, queue []botQueueRow) (response, error) {
	botIDs := make([]string, 0, len(queue))

	for _, bot := range queue {
		botIDs = append(botIDs, bot.BotID)
	}

	managers, err := impls.GetBotManagers(ctx, botIDs)

	if err != nil {
		return response{}, newError(err)
	}

	bots := make([]types.PartialEntity, 0, len(queue))

	for _, bot := range queue {
		// Dovewing is fronted by a Redis hot cache, so this stays per-bot.
		user, err := impls.GetPlatformUser(ctx, bot.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		bots = append(bots, types.PartialEntity{Bot: &types.PartialBot{
			BotID:                bot.BotID,
			ClientID:             bot.ClientID,
			User:                 user,
			ClaimedBy:            bot.ClaimedBy,
			LastClaimed:          types.TimestampPtr(bot.LastClaimed),
			ApprovalNote:         bot.ApprovalNote,
			Short:                bot.Short,
			Type:                 bot.Type,
			Votes:                bot.ApproximateVotes,
			Shards:               bot.Shards,
			Library:              bot.Library,
			InviteClicks:         bot.InviteClicks,
			Clicks:               bot.Clicks,
			Servers:              bot.Servers,
			Mentionable:          managers[bot.BotID].Mentionables(),
			Invite:               bot.Invite,
			ModerationFlagged:    bot.ModerationFlagged,
			ModerationCategories: types.NonNilStrings(bot.ModerationCategories),
		}})
	}

	return writeJSON(http.StatusOK, bots), nil
}
