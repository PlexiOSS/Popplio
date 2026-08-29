package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// queueSession is one live `queue` browser. Sessions are author-scoped and
// expire after 120 seconds, matching poise's interaction timeout.
type queueSession struct {
	AuthorID  string
	Bots      []queueBot
	Current   int
	TextMode  bool
	ExpiresAt time.Time
}

type queueBot struct {
	ClaimedBy    *string `db:"claimed_by"`
	BotID        string  `db:"bot_id"`
	ApprovalNote string  `db:"approval_note"`
	Short        string  `db:"short"`
	Invite       string  `db:"invite"`
	ClientID     string  `db:"client_id"`
}

var (
	sessionsMu sync.Mutex
	// queueSessions is keyed by the message id the buttons live on.
	queueSessions = map[string]*queueSession{}
	// claimSessions is keyed by message id for the claim force/remind prompt.
	claimSessions = map[string]*claimSession{}
	// rpcSessions is keyed by user id for the rpc modal driver.
	rpcSessions = map[string]*rpcSession{}
)

type claimSession struct {
	AuthorID  string
	BotID     string
	ClaimedBy string
}

type rpcSession struct {
	AuthorID   string
	TargetType types.TargetType
	Method     string
}

func cmdQueue() *Command {
	return &Command{
		Name:        "queue",
		Category:    "Testing",
		Description: "Shows the queue of bots pending review",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionBool{Name: "embed", Description: "Whether to embed or not"},
		},
		Run: func(c *Ctx) error {
			embed := c.BoolOption("embed", true)

			rows, err := db.New(state.Pool).GetPendingBotsQueue(c.Context)

			if err != nil {
				return err
			}

			bots := make([]queueBot, len(rows))

			for i, row := range rows {
				var claimedBy *string

				if row.ClaimedBy.Valid {
					claimedBy = &row.ClaimedBy.String
				}

				bots[i] = queueBot{
					ClaimedBy:    claimedBy,
					BotID:        row.BotID,
					ApprovalNote: row.ApprovalNote,
					Short:        row.Short,
					Invite:       row.Invite,
					ClientID:     row.ClientID,
				}
			}

			if len(bots) == 0 {
				return c.Say("There are no bots in the queue!")
			}

			session := &queueSession{
				AuthorID:  c.Author.ID.String(),
				Bots:      bots,
				Current:   0,
				TextMode:  !embed,
				ExpiresAt: time.Now().Add(120 * time.Second),
			}

			msg, err := renderQueue(c, session)

			if err != nil {
				return err
			}

			messageID, err := c.SendTracked(msg)

			if err != nil {
				return err
			}

			sessionsMu.Lock()
			queueSessions[messageID.String()] = session
			sessionsMu.Unlock()

			return nil
		},
	}
}

// renderQueue builds the message for the session's current position.
func renderQueue(c *Ctx, s *queueSession) (discord.MessageCreate, error) {
	bot := s.Bots[s.Current]

	owners, err := impls.GetEntityManagers(c.Context, types.TargetTypeBot, bot.BotID)

	if err != nil {
		return discord.MessageCreate{}, err
	}

	user, err := impls.GetPlatformUser(c.Context, bot.BotID)

	if err != nil {
		return discord.MessageCreate{}, err
	}

	claimedBy := "*You are free to test this bot. It is not claimed*"

	if bot.ClaimedBy != nil {
		claimedBy = *bot.ClaimedBy
	}

	msg := discord.MessageCreate{
		Components: queueComponents(s, bot),
	}

	if s.TextMode {
		msg.Content = fmt.Sprintf(
			"**%s [%d/%d]**\n**ID:** %s\n**Claimed by:** %s\n**Approval note:** %s\n**Short:** %s\n**Owner:** %s\n**Invite:** %s",
			user.DisplayName, s.Current+1, len(s.Bots), bot.BotID, claimedBy,
			bot.ApprovalNote, bot.Short, owners.MentionUsers(), bot.Invite)

		return msg, nil
	}

	msg.Embeds = []discord.Embed{{
		Title: fmt.Sprintf("%s %d/%d", user.DisplayName, s.Current+1, len(s.Bots)),
		Fields: []discord.EmbedField{
			{Name: "ID", Value: bot.BotID, Inline: impls.InlineFalse()},
			{Name: "Short", Value: bot.Short, Inline: impls.InlineFalse()},
			{Name: "Owner", Value: owners.MentionUsers(), Inline: impls.InlineFalse()},
			{Name: "Claimed by", Value: claimedBy, Inline: impls.InlineFalse()},
			{Name: "Approval note", Value: bot.ApprovalNote, Inline: impls.InlineTrue()},
			{Name: "Invite", Value: fmt.Sprintf("[Invite Bot](%s)", bot.Invite), Inline: impls.InlineTrue()},
		},
	}}

	return msg, nil
}

func queueComponents(s *queueSession, bot queueBot) []discord.ContainerComponent {
	safeInvite := fmt.Sprintf(
		"https://discord.com/api/v10/oauth2/authorize?client_id=%s&permissions=0&scope=bot%%20applications.commands&guild_id=%s",
		bot.ClientID, state.Config.Servers.Testing)

	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.NewPrimaryButton("Previous", "q:prev").WithDisabled(s.Current == 0),
			discord.NewDangerButton("Cancel", "q:cancel"),
			discord.NewPrimaryButton("Next", "q:next").WithDisabled(s.Current >= len(s.Bots)-1),
		},
		discord.ActionRowComponent{
			discord.NewLinkButton("Invite", safeInvite),
			discord.NewLinkButton("Invite (DB, Unsafe)", bot.Invite),
			discord.NewLinkButton("View Page", state.Config.Sites.Frontend+"/bots/"+bot.BotID),
		},
	}
}

func cmdClaim() *Command {
	return &Command{
		Name:        "claim",
		Category:    "Testing",
		Description: "Claims a bot",
		Checks:      []Check{isStaff, testingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "bot", Description: "The bot you wish to claim", Required: true},
		},
		Run: func(c *Ctx) error {
			botID := strings.Trim(c.Option("bot", 0), "<@!>")

			row, err := db.New(state.Pool).GetBotTypeAndClaimedBy(c.Context, botID)

			if err != nil {
				return err
			}

			if row.Type != "pending" {
				return fmt.Errorf("This bot is not pending review")
			}

			if row.ClaimedBy.Valid {
				claimedBy := row.ClaimedBy.String

				// Offer force-claim or a reminder rather than claiming outright.
				messageID, err := c.SendTracked(discord.MessageCreate{
					Embeds: []discord.Embed{{
						Title:       "Bot Already Claimed",
						Description: fmt.Sprintf("This bot is already claimed by <@%s>", claimedBy),
						Color:       impls.ColourRed,
					}},
					Components: []discord.ContainerComponent{
						discord.ActionRowComponent{
							discord.NewDangerButton("Force Claim", "fclaim"),
							discord.NewSecondaryButton("Remind Reviewer", "remind"),
						},
					},
				})

				if err != nil {
					return err
				}

				sessionsMu.Lock()
				claimSessions[messageID.String()] = &claimSession{
					AuthorID:  c.Author.ID.String(),
					BotID:     botID,
					ClaimedBy: claimedBy,
				}
				sessionsMu.Unlock()

				return nil
			}

			_, err = runRPC(c, types.RPCMethod{Claim: &types.RPCClaim{TargetID: botID, Force: false}})

			if err != nil {
				return err
			}

			return c.Ok("Claimed bot successfully, the bot owner has been informed")
		},
	}
}

func cmdUnclaim() *Command {
	return &Command{
		Name:        "unclaim",
		Category:    "Testing",
		Description: "Unclaims a bot",
		Checks:      []Check{isStaff, testingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "bot", Description: "The bot you wish to unclaim", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "Reason for unclaiming", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := c.Defer(); err != nil {
				return err
			}

			botID := strings.Trim(c.Option("bot", 0), "<@!>")
			reason := c.reasonArg("reason", 1)

			_, err := runRPC(c, types.RPCMethod{Unclaim: &types.RPCTargetReason{TargetID: botID, Reason: reason}})

			if err != nil {
				return err
			}

			return c.Ok("Unclaimed bot successfully!")
		},
	}
}

func cmdApprove() *Command {
	return &Command{
		Name:        "approve",
		Category:    "Testing",
		Description: "Approves a bot",
		Checks:      []Check{isStaff, testingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "bot", Description: "The bot you wish to approve", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "The reason for approval", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := c.Defer(); err != nil {
				return err
			}

			botID := strings.Trim(c.Option("bot", 0), "<@!>")
			reason := c.reasonArg("reason", 1)

			res, err := runRPC(c, types.RPCMethod{Approve: &types.RPCTargetReason{TargetID: botID, Reason: reason}})

			if err != nil {
				return err
			}

			content, ok := res.Text()

			if !ok {
				return fmt.Errorf("RPC did not return as expected???")
			}

			return c.Ok(fmt.Sprintf("Approved bot!\n%s", content))
		},
	}
}

func cmdDeny() *Command {
	return &Command{
		Name:        "deny",
		Category:    "Testing",
		Description: "Denies a bot",
		Checks:      []Check{isStaff, testingServer},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionUser{Name: "bot", Description: "The bot you wish to deny", Required: true},
			discord.ApplicationCommandOptionString{Name: "reason", Description: "The reason for denial", Required: true},
		},
		Run: func(c *Ctx) error {
			if err := c.Defer(); err != nil {
				return err
			}

			botID := strings.Trim(c.Option("bot", 0), "<@!>")
			reason := c.reasonArg("reason", 1)

			_, err := runRPC(c, types.RPCMethod{Deny: &types.RPCTargetReason{TargetID: botID, Reason: reason}})

			if err != nil {
				return err
			}

			return c.Ok("Okay! The bot has been denied.")
		},
	}
}

// reasonArg reads a reason, joining the remaining positional arguments so a
// prefix invocation can pass a multi-word reason without quoting.
func (c *Ctx) reasonArg(name string, from int) string {
	if v, ok := c.Options[name]; ok {
		return v
	}

	if from < len(c.Args) {
		return strings.Join(c.Args[from:], " ")
	}

	return ""
}

// claimReminderLog records that a reviewer was nudged.
func claimReminderLog(c *Ctx, botID, claimedBy string) error {
	return db.New(state.Pool).InsertStaffGeneralLog(c.Context, db.InsertStaffGeneralLogParams{
		UserID: c.Author.ID.String(),
		Action: "claim_reminder",
		Data: map[string]any{
			"bot_id":     botID,
			"claimed_by": claimedBy,
		},
	})
}
