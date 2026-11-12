// Copyright (C) 2026 NodeByte LTD

package autovote

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"popplio/audit"
	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/paymentmethod"
	"go.uber.org/zap"
)

const (
	MaxAccountsPerCard = 3

	SpikeMinAutoVotes = 10
	SpikeMinShare     = 0.5
	SpikeMinJump      = 0.25

	CardLimitSkipReason = "This payment card already pays for Auto-vote on 3 other accounts, so automatic votes on this account are paused. Contact support if this is a mistake."
)

func cardFingerprint(pm *stripe.PaymentMethod) string {
	if pm == nil {
		return ""
	}

	if pm.Card != nil && pm.Card.Fingerprint != "" {
		return pm.Card.Fingerprint
	}

	if pm.ID == "" {
		return ""
	}

	full, err := paymentmethod.Get(pm.ID, nil)

	if err != nil || full.Card == nil {
		return ""
	}

	return full.Card.Fingerprint
}

func RecordCard(ctx context.Context, userID string, sub *stripe.Subscription) {
	fingerprint := cardFingerprint(sub.DefaultPaymentMethod)

	if fingerprint == "" {
		return
	}

	q := db.New(state.Pool)

	changed, err := q.SetAutoVoteCardFingerprint(ctx, db.SetAutoVoteCardFingerprintParams{
		StripeSubscriptionID: sub.ID,
		CardFingerprint:      pgText(fingerprint),
	})

	if err != nil {
		state.Logger.Warn("Failed to save auto-vote card fingerprint", zap.Error(err), zap.String("subscription", sub.ID))
		return
	}

	if changed == 0 {
		return
	}

	accounts, err := q.ListAutoVoteAccountsOnCard(ctx, pgText(fingerprint))

	if err != nil {
		state.Logger.Warn("Failed to count auto-vote accounts on card", zap.Error(err), zap.String("subscription", sub.ID))
		return
	}

	position := slices.Index(accounts, userID)

	if position < MaxAccountsPerCard {
		return
	}

	audit.Record(audit.Event{
		Source:     audit.SourceSystem,
		ActorType:  audit.ActorUser,
		ActorID:    userID,
		Action:     "autovote.card_limit",
		TargetType: "user",
		TargetID:   userID,
		Data:       map[string]any{"subscription": sub.ID, "accounts_on_card": len(accounts)},
	})

	alertStaff(fmt.Sprintf(
		"One payment card now pays for Auto-vote on %d accounts: %s. Automatic votes are paused on every account after the first %d. Check whether these are the same person.",
		len(accounts), strings.Join(accounts, ", "), MaxAccountsPerCard,
	))
}

func IsShareSpike(autoDay, totalDay, autoBefore, totalBefore int) bool {
	if autoDay < SpikeMinAutoVotes || totalDay == 0 {
		return false
	}

	share := float64(autoDay) / float64(totalDay)
	before := 0.0

	if totalBefore > 0 {
		before = float64(autoBefore) / float64(totalBefore)
	}

	return share >= SpikeMinShare && share-before >= SpikeMinJump
}

func alertStaff(text string) {
	if state.Discord == nil || state.Config == nil || state.Config.Channels.StaffLogs == 0 {
		return
	}

	if _, err := state.Discord.Rest.CreateMessage(state.Config.Channels.StaffLogs, discord.MessageCreate{
		Embeds: []discord.Embed{{Title: "Auto-vote abuse check", Description: text}},
	}); err != nil {
		state.Logger.Warn("Failed to alert staff about auto-vote", zap.Error(err))
	}
}

func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
