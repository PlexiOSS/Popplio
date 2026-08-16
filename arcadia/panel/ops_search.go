package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"
	"popplio/votes"

	"github.com/jackc/pgx/v5"
)

// Entity search, one query shape per target type.

type searchServerRow struct {
	ServerID         string     `db:"server_id"`
	Name             string     `db:"name"`
	Avatar           string     `db:"avatar"`
	TotalMembers     int32      `db:"total_members"`
	OnlineMembers    int32      `db:"online_members"`
	Short            string     `db:"short"`
	Type             string     `db:"type"`
	ApproximateVotes int32      `db:"approximate_votes"`
	InviteClicks     int32      `db:"invite_clicks"`
	Clicks           int32      `db:"clicks"`
	NSFW             bool       `db:"nsfw"`
	Tags             []string   `db:"tags"`
	Premium          bool       `db:"premium"`
	ClaimedBy        *string    `db:"claimed_by"`
	LastClaimed      *time.Time `db:"last_claimed"`
}

func (s *Server) searchEntitys(ctx context.Context, q *types.QSearchEntitys) (response, error) {
	// No permission check: open to all staff.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	pattern := "%" + q.Query + "%"

	switch q.TargetType {
	case types.TargetTypeBot:
		rows, err := state.Pool.Query(ctx,
			`SELECT bot_id, client_id, type, approximate_votes, shards, library, invite_clicks, clicks,
                        servers, last_claimed, claimed_by, approval_note, short, invite FROM bots
                        INNER JOIN internal_user_cache__discord discord_users ON bots.bot_id = discord_users.id
                        WHERE bot_id = $1 OR client_id = $1 OR discord_users.username ILIKE $2 ORDER BY bots.created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[botQueueRow])

		if err != nil {
			return response{}, newError(err)
		}

		return s.partialBots(ctx, queue)
	case types.TargetTypeServer:
		rows, err := state.Pool.Query(ctx,
			`SELECT server_id, name, avatar, total_members, online_members, short, type, approximate_votes, invite_clicks,
                        clicks, nsfw, tags, premium, claimed_by, last_claimed FROM servers
                        WHERE server_id = $1 OR name ILIKE $2 ORDER BY created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[searchServerRow])

		if err != nil {
			return response{}, newError(err)
		}

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
				ServerID: server.ServerID,
				Name:     server.Name,
				// Populated from servers.avatar, synced by Infernoplex's
				// serversync task while it's a member of the guild. Empty
				// until the first sync (or if the bot has never joined).
				Avatar:        server.Avatar,
				TotalMembers:  server.TotalMembers,
				OnlineMembers: server.OnlineMembers,
				Short:         server.Short,
				Type:          server.Type,
				Votes:         server.ApproximateVotes,
				InviteClicks:  server.InviteClicks,
				Clicks:        server.Clicks,
				NSFW:          server.NSFW,
				Tags:          types.NonNilStrings(server.Tags),
				Premium:       server.Premium,
				ClaimedBy:     server.ClaimedBy,
				LastClaimed:   types.TimestampPtr(server.LastClaimed),
				Mentionable:   managers[server.ServerID].Mentionables(),
			}})
		}

		return writeJSON(http.StatusOK, servers), nil
	case types.TargetTypePack:
		rows, err := state.Pool.Query(ctx,
			`SELECT url, name, short, pack_type, owner, tags, vote_banned FROM packs
                        WHERE url = $1 OR name ILIKE $2 ORDER BY created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[searchPackRow])

		if err != nil {
			return response{}, newError(err)
		}

		packs := make([]types.PartialEntity, 0, len(queue))

		for _, pack := range queue {
			owner, err := impls.GetPlatformUser(ctx, pack.Owner)

			if err != nil {
				return response{}, newError(err)
			}

			voteCount, err := votes.EntityGetVoteCount(ctx, state.Pool, pack.URL, "pack")

			if err != nil {
				return response{}, newError(err)
			}

			packs = append(packs, types.PartialEntity{Pack: &types.PartialPack{
				URL:        pack.URL,
				Name:       pack.Name,
				Short:      pack.Short,
				PackType:   pack.PackType,
				Owner:      owner,
				Votes:      int32(voteCount),
				Tags:       types.NonNilStrings(pack.Tags),
				VoteBanned: pack.VoteBanned,
			}})
		}

		return writeJSON(http.StatusOK, packs), nil
	case types.TargetTypeTeam:
		rows, err := state.Pool.Query(ctx,
			`SELECT id, name, COALESCE(short, '') AS short, tags, nsfw, vote_banned FROM teams
                        WHERE id = $1 OR name ILIKE $2 ORDER BY created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[searchTeamRow])

		if err != nil {
			return response{}, newError(err)
		}

		teams := make([]types.PartialEntity, 0, len(queue))

		for _, team := range queue {
			voteCount, err := votes.EntityGetVoteCount(ctx, state.Pool, team.ID, "team")

			if err != nil {
				return response{}, newError(err)
			}

			teams = append(teams, types.PartialEntity{Team: &types.PartialTeam{
				ID:         team.ID,
				Name:       team.Name,
				Short:      team.Short,
				Votes:      int32(voteCount),
				Tags:       types.NonNilStrings(team.Tags),
				NSFW:       team.NSFW,
				VoteBanned: team.VoteBanned,
			}})
		}

		return writeJSON(http.StatusOK, teams), nil
	case types.TargetTypeUser:
		rows, err := state.Pool.Query(ctx,
			`SELECT users.user_id, users.banned, users.vote_banned FROM users
                        INNER JOIN internal_user_cache__discord discord_users ON users.user_id = discord_users.id
                        WHERE users.user_id = $1 OR discord_users.username ILIKE $2 ORDER BY users.created_at`,
			q.Query, pattern)

		if err != nil {
			return response{}, newError(err)
		}

		queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[searchUserRow])

		if err != nil {
			return response{}, newError(err)
		}

		results := make([]types.PartialEntity, 0, len(queue))

		for _, u := range queue {
			platformUser, err := impls.GetPlatformUser(ctx, u.UserID)

			if err != nil {
				return response{}, newError(err)
			}

			var isStaff bool

			err = state.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM staff_members WHERE user_id = $1)", u.UserID).Scan(&isStaff)

			if err != nil {
				return response{}, newError(err)
			}

			results = append(results, types.PartialEntity{User: &types.PartialUser{
				User:       platformUser,
				Staff:      isStaff,
				Banned:     u.Banned,
				VoteBanned: u.VoteBanned,
			}})
		}

		return writeJSON(http.StatusOK, results), nil
	default:
		return writeText(http.StatusNotImplemented, "Searching this target type is not implemented"), nil
	}
}

type searchPackRow struct {
	URL        string   `db:"url"`
	Name       string   `db:"name"`
	Short      string   `db:"short"`
	PackType   string   `db:"pack_type"`
	Owner      string   `db:"owner"`
	Tags       []string `db:"tags"`
	VoteBanned bool     `db:"vote_banned"`
}

type searchTeamRow struct {
	ID         string   `db:"id"`
	Name       string   `db:"name"`
	Short      string   `db:"short"`
	Tags       []string `db:"tags"`
	NSFW       bool     `db:"nsfw"`
	VoteBanned bool     `db:"vote_banned"`
}

type searchUserRow struct {
	UserID     string `db:"user_id"`
	Banned     bool   `db:"banned"`
	VoteBanned bool   `db:"vote_banned"`
}
