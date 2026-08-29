package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/db"
	"popplio/state"
	ptypes "popplio/types"
	"popplio/votes"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func BansSync(ctx context.Context) error {
	bans, err := fetchAllBans(state.Config.Servers.Main)

	if err != nil {
		return fmt.Errorf("Error while fetching bans: %s", err)
	}

	q := db.New(state.Pool)

	dbBanned, err := q.GetBannedUserIDs(ctx)

	if err != nil {
		return fmt.Errorf("Error while fetching bans from database: %s", err)
	}

	var pingUsers strings.Builder

	for _, owner := range state.Config.Arcadia.Owners {
		fmt.Fprintf(&pingUsers, "<@%s>", owner)
	}

	serverBans := make(map[string]struct{}, len(bans))

	for _, ban := range bans {
		serverBans[ban.User.ID.String()] = struct{}{}
	}

	dbBans := make(map[string]struct{}, len(dbBanned))

	for _, userID := range dbBanned {
		dbBans[userID] = struct{}{}
	}

	toModify := make([]string, 0)

	for userID := range serverBans {
		if _, ok := dbBans[userID]; !ok {
			toModify = append(toModify, userID)
		}
	}

	for userID := range dbBans {
		if _, ok := serverBans[userID]; !ok {
			toModify = append(toModify, userID)
		}
	}

	state.Logger.Warn("Bans to modify", zap.Strings("users", toModify))

	for _, userID := range toModify {
		_, isBanned := serverBans[userID]

		rowsAffected, err := q.UpdateUserBanned(ctx, db.UpdateUserBannedParams{UserID: userID, Banned: isBanned})

		if err != nil {
			return fmt.Errorf("Error while updating user %s in database: %v", userID, err)
		}

		if rowsAffected == 0 {
			err := q.InsertBannedUser(ctx, db.InsertBannedUserParams{
				UserID:   userID,
				Banned:   isBanned,
				ApiToken: impls.GenRandom(512),
			})

			if err != nil {
				return fmt.Errorf("Error while inserting user %s into database: %v", userID, err)
			}
		}

		title := "User Unban"
		description := fmt.Sprintf("User %s was unbanned", userID)
		colour := impls.ColourBlurple

		if isBanned {
			title = "User Ban"
			description = fmt.Sprintf("User %s was banned", userID)
			colour = impls.ColourRed
		}

		err = impls.SendModLog(discord.MessageCreate{
			Content: pingUsers.String(),
			Embeds: []discord.Embed{{
				Title:       title,
				Description: description,
				Color:       colour,
			}},
		})

		if err != nil {
			return err
		}

		alertTitle := "Account Unbanned"
		alertMessage := "Your account has been unbanned."
		alertType := ptypes.AlertTypeSuccess

		if isBanned {
			alertTitle = "Account Banned"
			alertMessage = "Your account has been banned. If you believe this is a mistake, you can appeal."
			alertType = ptypes.AlertTypeError
		}

		impls.NotifyOwners([]string{userID}, ptypes.Alert{
			Type:     alertType,
			Title:    alertTitle,
			Message:  alertMessage,
			URL:      pgtype.Text{String: state.Config.Sites.Frontend + "/banned", Valid: true},
			Category: ptypes.AlertCategoryAccountSecurity,
		})
	}

	return nil
}

func fetchAllBans(guildID snowflake.ID) ([]discord.Ban, error) {
	var (
		all   []discord.Ban
		after snowflake.ID
	)

	for {
		page, err := dclient.Get().Rest().GetBans(guildID, 0, after, 1000)

		if err != nil {
			return nil, err
		}

		all = append(all, page...)

		if len(page) < 1000 {
			return all, nil
		}

		after = page[len(page)-1].User.ID
	}
}

func SpecRoleSync(ctx context.Context) error {
	if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Main); !ok {
		return fmt.Errorf("Failed to get guild")
	}

	var bugHunters []string

	dclient.Get().Caches().MembersForEach(state.Config.Servers.Main, func(member discord.Member) {
		for _, roleID := range member.RoleIDs {
			if roleID == state.Config.Roles.BugHunters {
				bugHunters = append(bugHunters, member.User.ID.String())
				return
			}
		}
	})

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	txq := db.New(tx)

	if err := txq.ClearAllBugHunters(ctx); err != nil {
		return fmt.Errorf("Error updating users: %v", err)
	}

	for _, userID := range bugHunters {
		if err := txq.SetUserBugHunter(ctx, userID); err != nil {
			return fmt.Errorf("Error updating users: %v", err)
		}
	}

	return tx.Commit(ctx)
}

type AutomatedVoteResetRow struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
}

func VoteResetter(ctx context.Context) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	txq := db.New(tx)

	_, err = txq.GetRecentAutomatedVoteResetForUpdate(ctx)

	if err == nil {
		return nil
	}

	if !isNoRows(err) {
		return err
	}

	if err := txq.LockEntityVotesExclusive(ctx); err != nil {
		return err
	}

	err = txq.VoidAllUnvoidedEntityVotes(ctx)

	if err != nil {
		return err
	}

	for _, targetType := range []string{"bot", "server", "team"} {
		if err := votes.RecomputeApproximateVotes(ctx, tx, targetType); err != nil {
			return err
		}
	}

	if err := txq.InsertAutomatedVoteReset(ctx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:  "__Automated Per-Monthly Vote Reset!__",
			Footer: impls.Footer("Welcome back :)"),
			Color:  impls.ColourRed,
		}},
	})
}

func isNoRows(err error) bool {
	return err != nil && err.Error() == pgx.ErrNoRows.Error()
}

type TopReviewer struct {
	UserID        string `db:"user_id"`
	ApprovedCount int64  `db:"approved_count"`
	DeniedCount   int64  `db:"denied_count"`
	TotalCount    int64  `db:"total_count"`
}

func QueryTopReviewers(ctx context.Context, limit int) ([]TopReviewer, error) {
	rows, err := db.New(state.Pool).GetTopReviewers(ctx, int32(limit))

	if err != nil {
		return nil, err
	}

	reviewers := make([]TopReviewer, len(rows))
	for i, row := range rows {
		reviewers[i] = TopReviewer{
			UserID:        row.UserID,
			ApprovedCount: row.ApprovedCount,
			DeniedCount:   row.DeniedCount,
			TotalCount:    row.TotalCount,
		}
	}

	return reviewers, nil
}

func TopReviewerSync(ctx context.Context) error {
	const limit = 3

	stats, err := QueryTopReviewers(ctx, limit)

	if err != nil {
		return err
	}

	if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Main); !ok {
		state.Logger.Error("Failed to get guild")
		return nil
	}

	if err := SyncTopReviewerRoles(ctx, stats); err != nil {
		return err
	}

	return impls.SendModLog(discord.MessageCreate{
		Content: "**Weekly Job**\nSynced Top Reviewers!",
	})
}

func SyncTopReviewerRoles(ctx context.Context, stats []TopReviewer) error {
	const reason = "Syncing top reviewers, weekly job."

	var holders []snowflake.ID

	dclient.Get().Caches().MembersForEach(state.Config.Servers.Main, func(member discord.Member) {
		for _, roleID := range member.RoleIDs {
			if roleID == state.Config.Roles.TopReviewers {
				holders = append(holders, member.User.ID)
				return
			}
		}
	})

	for _, userID := range holders {
		if err := impls.RemoveRole(state.Config.Servers.Main, userID, state.Config.Roles.TopReviewers, reason); err != nil {
			state.Logger.Error("Failed to remove role from member", zap.String("userID", userID.String()), zap.Error(err))
		}
	}

	for _, stat := range stats {
		userID, err := snowflake.Parse(stat.UserID)

		if err != nil {
			state.Logger.Error("Failed to parse user_id", zap.String("userID", stat.UserID))
			continue
		}

		if !impls.MemberOnGuild(state.Config.Servers.Main, userID) {
			continue
		}

		if err := impls.AddRole(state.Config.Servers.Main, userID, state.Config.Roles.TopReviewers, reason); err != nil {
			state.Logger.Error("Failed to add role to user", zap.String("userID", stat.UserID), zap.Error(err))
		}
	}

	return nil
}
