// Copyright (C) 2026 NodeByte LTD

package badgerules

import (
	"context"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type Rule struct {
	Type   string `json:"type"`
	Value  int64  `json:"value,omitempty"`
	RoleID string `json:"role_id,omitempty"`
}

type Kind struct {
	Type        string   `json:"type"`
	Category    string   `json:"category"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Targets     []string `json:"targets"`
	ValueLabel  string   `json:"value_label,omitempty"`
	MinValue    int64    `json:"min_value,omitempty"`
	NeedsRole   bool     `json:"needs_role,omitempty"`
}

const counted = `ev.upvote = true
	AND ev.void_reason IS DISTINCT FROM 'Votes (single entity) reset'
	AND NOT EXISTS (SELECT 1 FROM users u WHERE u.user_id = ev.author AND (u.vote_banned OR u.banned))`

const listed = `(type = 'approved' OR type = 'certified')`

const userListings = `
	SELECT owner AS user_id, bot_id AS id, type FROM bots WHERE owner IS NOT NULL AND ` + listed + `
	UNION
	SELECT tm.user_id, b.bot_id, b.type FROM bots b INNER JOIN team_members tm ON tm.team_id = b.team_owner WHERE ` + listed + `
	UNION
	SELECT tm.user_id, s.server_id, s.type FROM servers s INNER JOIN team_members tm ON tm.team_id = s.team_owner WHERE (s.type = 'approved' OR s.type = 'certified')`

const teamListings = `
	SELECT team_owner::text AS team_id, bot_id AS id, type FROM bots WHERE team_owner IS NOT NULL AND ` + listed + `
	UNION
	SELECT team_owner::text, server_id, type FROM servers WHERE ` + listed

var Kinds = []Kind{
	{Type: "listed_days", Category: "Listing", Label: "Listed for N days", Description: "Approved or certified for at least this many days.", Targets: []string{"bot", "server"}, ValueLabel: "Days", MinValue: 1},
	{Type: "total_votes", Category: "Listing", Label: "Has N votes", Description: "At least this many counted upvotes, all time.", Targets: []string{"bot", "server", "team"}, ValueLabel: "Votes", MinValue: 1},
	{Type: "certified", Category: "Listing", Label: "Is certified", Description: "The listing is certified.", Targets: []string{"bot", "server"}},
	{Type: "premium", Category: "Listing", Label: "Has premium", Description: "The listing currently has premium.", Targets: []string{"bot", "server"}},
	{Type: "partner", Category: "Listing", Label: "Is a partner", Description: "The bot is linked to an Omniplex partner.", Targets: []string{"bot"}},
	{Type: "votes_cast", Category: "Voting", Label: "Cast N votes", Description: "Has cast at least this many counted upvotes, all time.", Targets: []string{"user"}, ValueLabel: "Votes", MinValue: 1},
	{Type: "monthly_top", Category: "Voting", Label: "Top N voter in a month", Description: "Finished in the top N voters of any completed calendar month.", Targets: []string{"user"}, ValueLabel: "Top N", MinValue: 1},
	{Type: "active_months", Category: "Voting", Label: "Voted in N months", Description: "Has voted in at least this many different calendar months.", Targets: []string{"user"}, ValueLabel: "Months", MinValue: 1},
	{Type: "owns_listings", Category: "Ownership", Label: "Owns N listings", Description: "Owns, directly or through a team, at least this many approved or certified bots and servers.", Targets: []string{"user", "team"}, ValueLabel: "Listings", MinValue: 1},
	{Type: "owns_certified", Category: "Ownership", Label: "Owns a certified listing", Description: "Owns, directly or through a team, a certified bot or server.", Targets: []string{"user", "team"}},
	{Type: "reviews_written", Category: "Ownership", Label: "Wrote N reviews", Description: "Has written at least this many reviews (not replies).", Targets: []string{"user"}, ValueLabel: "Reviews", MinValue: 1},
	{Type: "server_booster", Category: "Discord", Label: "Boosts the Omniplex server", Description: "Currently boosting the main Omniplex Discord server.", Targets: []string{"user"}},
	{Type: "has_role", Category: "Discord", Label: "Has a role in the Omniplex server", Description: "Has this role in the main Omniplex Discord server.", Targets: []string{"user"}, NeedsRole: true},
}

var queries = map[string]map[string]string{
	"listed_days": {
		"bot":    `SELECT bot_id FROM bots WHERE ` + listed + ` AND created_at < NOW() - make_interval(days => $1::int)`,
		"server": `SELECT server_id FROM servers WHERE ` + listed + ` AND created_at < NOW() - make_interval(days => $1::int)`,
	},
	"total_votes": {
		"bot":    `SELECT ev.target_id FROM entity_votes ev INNER JOIN bots b ON b.bot_id = ev.target_id WHERE ev.target_type = 'bot' AND ` + counted + ` GROUP BY ev.target_id HAVING COUNT(*) >= $1`,
		"server": `SELECT ev.target_id FROM entity_votes ev INNER JOIN servers s ON s.server_id = ev.target_id WHERE ev.target_type = 'server' AND ` + counted + ` GROUP BY ev.target_id HAVING COUNT(*) >= $1`,
		"team":   `SELECT ev.target_id FROM entity_votes ev WHERE ev.target_type = 'team' AND ` + counted + ` GROUP BY ev.target_id HAVING COUNT(*) >= $1`,
	},
	"certified": {
		"bot":    `SELECT bot_id FROM bots WHERE type = 'certified'`,
		"server": `SELECT server_id FROM servers WHERE type = 'certified'`,
	},
	"premium": {
		"bot":    `SELECT bot_id FROM bots WHERE premium = true AND ` + listed,
		"server": `SELECT server_id FROM servers WHERE premium = true AND ` + listed,
	},
	"partner": {
		"bot": `SELECT DISTINCT b.bot_id FROM partners p INNER JOIN bots b ON b.bot_id = p.bot_id WHERE b.type = 'approved' OR b.type = 'certified'`,
	},
	"votes_cast": {
		"user": `SELECT ev.author FROM entity_votes ev WHERE ` + counted + ` GROUP BY ev.author HAVING COUNT(*) >= $1`,
	},
	"monthly_top": {
		"user": `WITH monthly AS (
			SELECT ev.author, date_trunc('month', ev.created_at) AS month, COUNT(*) AS votes, MIN(ev.created_at) AS first_vote
			FROM entity_votes ev WHERE ` + counted + ` AND ev.created_at < date_trunc('month', NOW())
			GROUP BY ev.author, date_trunc('month', ev.created_at)
		), ranked AS (
			SELECT author, ROW_NUMBER() OVER (PARTITION BY month ORDER BY votes DESC, first_vote ASC, author ASC) AS rank FROM monthly
		)
		SELECT DISTINCT author FROM ranked WHERE rank <= $1`,
	},
	"active_months": {
		"user": `SELECT ev.author FROM entity_votes ev WHERE ` + counted + ` GROUP BY ev.author HAVING COUNT(DISTINCT date_trunc('month', ev.created_at)) >= $1`,
	},
	"owns_listings": {
		"user": `SELECT user_id FROM (` + userListings + `) l GROUP BY user_id HAVING COUNT(DISTINCT id) >= $1`,
		"team": `SELECT team_id FROM (` + teamListings + `) l GROUP BY team_id HAVING COUNT(DISTINCT id) >= $1`,
	},
	"owns_certified": {
		"user": `SELECT DISTINCT user_id FROM (` + userListings + `) l WHERE type = 'certified'`,
		"team": `SELECT DISTINCT team_id FROM (` + teamListings + `) l WHERE type = 'certified'`,
	},
	"reviews_written": {
		"user": `SELECT author FROM reviews WHERE parent_id IS NULL AND owner_review = false GROUP BY author HAVING COUNT(*) >= $1`,
	},
}

var ErrDiscordNotReady = errors.New("the Discord member cache for the main server is empty")

func KindOf(t string) (Kind, bool) {
	for _, k := range Kinds {
		if k.Type == t {
			return k, true
		}
	}
	return Kind{}, false
}

func (k Kind) Supports(targetType string) bool {
	for _, t := range k.Targets {
		if t == targetType {
			return true
		}
	}
	return false
}

func Validate(r *Rule, targetTypes []string) error {
	if r == nil {
		return nil
	}

	k, ok := KindOf(r.Type)

	if !ok {
		return fmt.Errorf("unknown rule type %q", r.Type)
	}

	supported := false

	for _, t := range targetTypes {
		if k.Supports(t) {
			supported = true
		}
	}

	if !supported {
		return fmt.Errorf("the %q rule only applies to %v, and this badge can't be given to any of those", k.Label, k.Targets)
	}

	if k.ValueLabel != "" && r.Value < k.MinValue {
		return fmt.Errorf("%s must be at least %d", k.ValueLabel, k.MinValue)
	}

	if k.ValueLabel == "" {
		r.Value = 0
	}

	if k.NeedsRole {
		if _, err := snowflake.Parse(r.RoleID); err != nil {
			return errors.New("a valid role ID is required")
		}
	} else {
		r.RoleID = ""
	}

	return nil
}

func Qualifying(ctx context.Context, r Rule, targetType string) ([]string, error) {
	switch r.Type {
	case "server_booster", "has_role":
		if targetType != "user" {
			return []string{}, nil
		}
		return discordMembers(r)
	}

	sql, ok := queries[r.Type][targetType]

	if !ok {
		return []string{}, nil
	}

	var args []any

	if k, _ := KindOf(r.Type); k.ValueLabel != "" {
		args = append(args, r.Value)
	}

	rows, err := state.Pool.Query(ctx, sql, args...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ids := []string{}

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func discordMembers(r Rule) ([]string, error) {
	guild := state.Config.Servers.Main
	roleID, _ := snowflake.Parse(r.RoleID)

	ids := []string{}
	seen := 0

	state.Discord.Caches().MembersForEach(guild, func(m discord.Member) {
		seen++

		if m.User.Bot {
			return
		}

		switch r.Type {
		case "server_booster":
			if m.PremiumSince != nil {
				ids = append(ids, m.User.ID.String())
			}
		case "has_role":
			for _, id := range m.RoleIDs {
				if id == roleID {
					ids = append(ids, m.User.ID.String())
					return
				}
			}
		}
	})

	if seen == 0 {
		return nil, ErrDiscordNotReady
	}

	return ids, nil
}
