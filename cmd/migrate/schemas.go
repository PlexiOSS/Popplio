package main

import (
	"encoding/json"
	"time"

	"popplio/arcadia/bot"
	"popplio/arcadia/impls"
	"popplio/arcadia/panel"
	"popplio/arcadia/tasks"
	"popplio/perms"
	"popplio/types"
	"popplio/validators"

	"github.com/jackc/pgx/v5/pgtype"
)

type StaffDisciplinaryRow struct {
	ID          pgtype.UUID     `db:"id"`
	UserID      string          `db:"user_id"`
	CreatedAt   time.Time       `db:"created_at"`
	Expiry      pgtype.Interval `db:"expiry"`
	Title       string          `db:"title"`
	Description string          `db:"description"`
	Type        string          `db:"type"`
	State       string          `db:"state"`
}

type StaffGeneralLogRow struct {
	UserID    string          `db:"user_id"`
	Action    string          `db:"action"`
	Data      json.RawMessage `db:"data"`
	CreatedAt time.Time       `db:"created_at"`
}

type StaffOnboardingRow struct {
	ID              pgtype.UUID        `db:"id"`
	UserID          string             `db:"user_id"`
	State           string             `db:"state"`
	CreatedAt       time.Time          `db:"created_at"`
	FinishedAt      pgtype.Timestamptz `db:"finished_at"`
	GuildID         string             `db:"guild_id"`
	Void            bool               `db:"void"`
	Questions       json.RawMessage    `db:"questions"`
	Answers         json.RawMessage    `db:"answers"`
	Verdict         json.RawMessage    `db:"verdict"`
	StaffVerifyCode pgtype.Text        `db:"staff_verify_code"`
}

type ShopHoldRow struct {
	ID         pgtype.UUID     `db:"id"`
	TargetID   string          `db:"target_id"`
	TargetType string          `db:"target_type"`
	Item       string          `db:"item"`
	CreatedAt  time.Time       `db:"created_at"`
	Duration   pgtype.Interval `db:"duration"`
}

type NotificationPrefRow struct {
	UserID   string `db:"user_id"`
	Category string `db:"category"`
	Enabled  bool   `db:"enabled"`
}

type BadgeRow struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Icon        string    `db:"icon"`
	Color       string    `db:"color"`
	TargetTypes []string  `db:"target_types"`
	CreatedAt   time.Time `db:"created_at"`
	CreatedBy   string    `db:"created_by"`
	LastUpdated time.Time `db:"last_updated"`
	UpdatedBy   string    `db:"updated_by"`
}

type EntityBadgeRow struct {
	Itag       pgtype.UUID `db:"itag"`
	TargetType string      `db:"target_type"`
	TargetID   string      `db:"target_id"`
	BadgeID    string      `db:"badge_id"`
	Reason     string      `db:"reason"`
	AwardedBy  string      `db:"awarded_by"`
	CreatedAt  time.Time   `db:"created_at"`
}

var tableSchemas = map[string]any{
	"alerts":                       types.Alert{},
	"badges":                       BadgeRow{},
	"entity_badges":                EntityBadgeRow{},
	"shop_holds":                   ShopHoldRow{},
	"staff_disciplinary":           StaffDisciplinaryRow{},
	"staff_general_logs":           StaffGeneralLogRow{},
	"staff_onboardings":            StaffOnboardingRow{},
	"user_notification_prefs":      NotificationPrefRow{},
	"automated_vote_resets":        tasks.AutomatedVoteResetRow{},
	"blacklisted_words":            validators.BlacklistedWordRow{},
	"bot_whitelist":                panel.BotWhitelistRow{},
	"internal_user_cache__discord": perms.InternalUserCacheDiscordRow{},
	"mod_cases":                    bot.ModCaseRow{},
	"reports":                      panel.ReportRow{},
	"rpc_logs":                     panel.RPCLogRow{},
	"staff_disciplinary_types":     panel.DisciplinaryTypeRow{},
	"staff_members":                impls.StaffMemberRow{},
	"staff_positions":              impls.StaffPositionRow{},
	"staffpanel__authchain":        impls.StaffAuthChainRow{},
	"apps":                         types.AppResponse{},
	"api_sessions":                 types.Session{},
	"blogs":                        types.BlogPost{},
	"bot_changelogs":               types.BotChangelog{},
	"bot_commands":                 types.BotCommand{},
	"bots":                         types.Bot{},
	"changelogs":                   types.ChangelogEntry{},
	"entity_vote_redeem_logs":      types.EntityVoteRedeemLog{},
	"entity_votes":                 types.EntityVote{},
	"pack_emojis":                  types.PackEmoji{},
	"packs":                        types.BotPack{},
	"partner_types":                types.PartnerTypes{},
	"partners":                     types.Partner{},
	"reviews":                      types.Review{},
	"servers":                      types.Server{},
	"shop_coupons":                 types.ShopCoupon{},
	"shop_item_benefits":           types.ShopItemBenefit{},
	"shop_items":                   types.ShopItem{},
	"shop_purchases":               types.ShopPurchase{},
	"staff_template_types":         types.StaffTemplateType{},
	"staff_templates":              types.StaffTemplate{},
	"tasks":                        types.Task{},
	"team_members":                 types.TeamMember{},
	"teams":                        types.Team{},
	"tickets":                      types.Ticket{},
	"user_notifications":           types.NotifGet{},
	"user_reminders":               types.Reminder{},
	"users":                        types.User{},
	"vanity":                       types.Vanity{},
	"vote_credit_tiers":            types.VoteCreditTier{},
	"webhook_logs":                 types.WebhookLogEntry{},
	"webhooks":                     types.Webhook{},
}
