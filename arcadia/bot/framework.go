// Package bot is the staff Discord bot: the other half of the system that
// shares the RPC action layer with the panel.
//
// Arcadia used poise, which supplies a command framework (prefix + slash
// parsing, checks, cooldowns, help rendering, modals, pagination). disgo has no
// equivalent, so the pieces the commands actually rely on are implemented here.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/cmdsync"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// Ctx is what a command handler receives. It abstracts over a prefix invocation
// and a slash invocation so each command is written once.
type Ctx struct {
	Context context.Context

	// Author is who invoked the command.
	Author discord.User
	// GuildID is 0 in DMs.
	GuildID snowflake.ID
	// ChannelID is where the command was invoked.
	ChannelID snowflake.ID

	// Args are the positional arguments of a prefix invocation.
	Args []string
	// Options are the named options of a slash invocation.
	Options map[string]string

	// bot is the bot the command was invoked through.
	bot *Bot

	// slash is set for slash invocations.
	slash *events.ApplicationCommandInteractionCreate
	// replied tracks whether the interaction has been answered.
	replied bool
}

// Discord performs actions as the bot the command was invoked through.
func (c *Ctx) Discord() impls.Discord {
	return impls.With(c.client())
}

func (c *Ctx) client() bot.Client {
	if c.bot == nil {
		return dclient.Get()
	}

	return c.bot.client()
}

// Option reads a named option, falling back to the positional prefix argument at
// the given index.
func (c *Ctx) Option(name string, index int) string {
	if v, ok := c.Options[name]; ok {
		return v
	}

	if index >= 0 && index < len(c.Args) {
		return c.Args[index]
	}

	return ""
}

// BoolOption reads a boolean option, defaulting when absent.
func (c *Ctx) BoolOption(name string, fallback bool) bool {
	v, ok := c.Options[name]

	if !ok {
		return fallback
	}

	switch strings.ToLower(v) {
	case "true", "t", "y", "yes":
		return true
	case "false", "f", "n", "no":
		return false
	default:
		return fallback
	}
}

// Say replies with a message in the bot's neutral colour.
//
// Arcadia's replies are embeds, including one-liners: a bare content message
// is indistinguishable from a staff member talking, which matters in the staff
// server where the bot's answers and the conversation share a channel. Omi
// answers the public in plain text, and its refusals are only shown to the
// person who ran the command. The text itself is the same either way, so the
// strings frozen in arcadia/conformance still read exactly as they did.
func (c *Ctx) Say(content string) error {
	return c.SayColour(content, impls.ColourBlurple)
}

// Ok replies to something that worked.
func (c *Ctx) Ok(content string) error {
	return c.SayColour(content, impls.ColourGreen)
}

// Fail replies to something that did not.
//
// This is what the command guards and the panic handler use, so a refusal is
// visibly different from an answer without either of them having to say so.
func (c *Ctx) Fail(content string) error {
	return c.SayColour(content, impls.ColourRed)
}

// SayColour replies with a message in an explicit colour.
func (c *Ctx) SayColour(content string, colour int) error {
	if c.bot == omiBot {
		msg := discord.MessageCreate{
			Content:         content,
			AllowedMentions: &discord.AllowedMentions{},
			Flags:           discord.MessageFlagSuppressEmbeds,
		}

		if colour == impls.ColourRed && c.slash != nil {
			msg.Flags |= discord.MessageFlagEphemeral
		}

		return c.Send(msg)
	}

	return c.Send(discord.MessageCreate{Embeds: []discord.Embed{{
		Description: content,
		Color:       colour,
	}}})
}

// Send sends a full message reply.
func (c *Ctx) Send(msg discord.MessageCreate) error {
	if c.slash != nil {
		if c.replied {
			_, err := c.slash.Client().Rest().CreateFollowupMessage(
				c.slash.ApplicationID(), c.slash.Token(), msg)
			return err
		}

		c.replied = true
		return c.slash.CreateMessage(msg)
	}

	_, err := c.client().Rest().CreateMessage(c.ChannelID, msg)
	return err
}

// SendTracked sends a reply and returns the id of the resulting message.
//
// The component-driven commands (queue, claim) key their session on that id. A
// slash invocation must be answered through the interaction or Discord shows the
// caller "the application did not respond", so this replies the right way for
// the invocation and then resolves the message id either way.
func (c *Ctx) SendTracked(msg discord.MessageCreate) (snowflake.ID, error) {
	if c.slash == nil {
		sent, err := c.client().Rest().CreateMessage(c.ChannelID, msg)

		if err != nil {
			return 0, err
		}

		return sent.ID, nil
	}

	if err := c.Send(msg); err != nil {
		return 0, err
	}

	sent, err := c.client().Rest().GetInteractionResponse(c.slash.ApplicationID(), c.slash.Token())

	if err != nil {
		return 0, err
	}

	return sent.ID, nil
}

// Defer acknowledges a slash command that will take a while. It is a no-op for
// prefix invocations.
func (c *Ctx) Defer() error {
	if c.slash == nil || c.replied {
		return nil
	}

	c.replied = true
	return c.slash.DeferCreateMessage(false)
}

// Check is a command guard. Returning an error surfaces its text to the user,
// which is how "You are not staff" reaches them.
type Check func(c *Ctx) error

// Command is one bot command.
type Command struct {
	Name        string
	Category    string
	Description string
	// Options are the slash-command options, in order.
	Options      []discord.ApplicationCommandOption
	Checks       []Check
	Run          func(c *Ctx) error
	Autocomplete func(c *Ctx, value string) []discord.AutocompleteChoice
	// Subcommands are dispatched on the first positional argument.
	Subcommands []*Command
	// OwnerOnly restricts the command to the configured owners.
	OwnerOnly bool
}

// Bot is one Discord bot that runs commands from this package. Omi runs the
// public commands in the main and testing servers, and Arcadia runs the staff
// commands in the staff and testing servers.
type Bot struct {
	Name   string
	client func() bot.Client
	guilds func() []snowflake.ID
	prefix func() string

	registry   map[string]*Command
	ordered    []*Command
	categories []string
}

func newBot(name string, client func() bot.Client, guilds func() []snowflake.ID, prefix func() string) *Bot {
	return &Bot{Name: name, client: client, guilds: guilds, prefix: prefix, registry: map[string]*Command{}}
}

var arcadiaBot = newBot("arcadia", dclient.Get,
	func() []snowflake.ID { return []snowflake.ID{state.Config.Servers.Staff, state.Config.Servers.Testing} },
	func() string {
		if state.Config.Arcadia.PrefixCommands {
			return state.Config.Arcadia.Prefix
		}

		return "/"
	},
)

var omiBot = newBot("omi", func() bot.Client { return state.Discord },
	func() []snowflake.ID { return []snowflake.ID{state.Config.Servers.Main, state.Config.Servers.Testing} },
	func() string { return "/" },
)

func (b *Bot) register(cmds ...*Command) {
	for _, cmd := range cmds {
		b.registry[cmd.Name] = cmd
		b.ordered = append(b.ordered, cmd)
	}
}

func (b *Bot) reset() {
	b.registry = map[string]*Command{}
	b.ordered = nil
	b.categories = nil
}

var registerOnce sync.Once

func ensureRegistered() {
	registerOnce.Do(func() {
		registerCommands()
		arcadiaBot.indexCategories()
		omiBot.indexCategories()
	})
}

// Listener returns Arcadia's event listener. It is built before the gateway
// connects so no early event is missed; the caller hands it to dclient.Setup.
func Listener(ctx context.Context) bot.EventListener {
	ensureRegistered()

	adapter := &events.ListenerAdapter{
		OnGuildsReady:                   func(e *events.GuildsReady) { onGuildsReady(ctx, e) },
		OnApplicationCommandInteraction: func(e *events.ApplicationCommandInteractionCreate) { onSlashCommand(ctx, arcadiaBot, e) },
		OnComponentInteraction:          func(e *events.ComponentInteractionCreate) { onComponent(ctx, e) },
		OnModalSubmit:                   func(e *events.ModalSubmitInteractionCreate) { onModalSubmit(ctx, e) },
	}

	if state.Config.Arcadia.PrefixCommands {
		adapter.OnMessageCreate = func(e *events.MessageCreate) { onMessageCreate(ctx, e) }
	}

	return adapter
}

// OmiListener returns the event listener added to Omi's client: the public
// commands, auto-mod and the main server's join announcements.
func OmiListener(ctx context.Context) bot.EventListener {
	ensureRegistered()

	adapter := &events.ListenerAdapter{
		OnGuildsReady: func(*events.GuildsReady) {
			if err := SyncOmiCommands(); err != nil {
				state.Logger.Error("Failed to sync Omi's application commands", zap.Error(err))
			}
		},
		OnGuildMemberJoin:               onGuildMemberJoin,
		OnApplicationCommandInteraction: func(e *events.ApplicationCommandInteractionCreate) { onSlashCommand(ctx, omiBot, e) },
		OnAutocompleteInteraction:       func(e *events.AutocompleteInteractionCreate) { onAutocomplete(ctx, omiBot, e) },
		OnComponentInteraction: func(e *events.ComponentInteractionCreate) {
			switch id := e.Data.CustomID(); {
			case strings.HasPrefix(id, "kb:"):
				onKbComponent(ctx, e)
			case strings.HasPrefix(id, voteBoardPrefix):
				onVoteBoardComponent(ctx, e)
			}
		},
	}

	if state.Config.Arcadia.AutoMod {
		adapter.OnMessageCreate = func(e *events.MessageCreate) {
			if !e.Message.Author.Bot {
				scanMessage(ctx, e)
			}
		}
	}

	return adapter
}

func onMessageCreate(ctx context.Context, e *events.MessageCreate) {
	if e.Message.Author.Bot {
		return
	}

	p := arcadiaBot.prefix()

	if !strings.HasPrefix(e.Message.Content, p) {
		return
	}

	fields := strings.Fields(strings.TrimPrefix(e.Message.Content, p))

	if len(fields) == 0 {
		return
	}

	cmd, ok := arcadiaBot.registry[strings.ToLower(fields[0])]

	if !ok {
		return
	}

	c := &Ctx{
		Context:   ctx,
		Author:    e.Message.Author,
		ChannelID: e.ChannelID,
		Args:      fields[1:],
		Options:   map[string]string{},
		bot:       arcadiaBot,
	}

	if e.GuildID != nil {
		c.GuildID = *e.GuildID
	}

	invoke(cmd, c)
}

func onSlashCommand(ctx context.Context, b *Bot, e *events.ApplicationCommandInteractionCreate) {
	data := e.SlashCommandInteractionData()

	cmd, ok := b.registry[data.CommandName()]

	if !ok {
		return
	}

	c := &Ctx{
		Context:   ctx,
		Author:    e.User(),
		ChannelID: e.Channel().ID(),
		Options:   map[string]string{},
		bot:       b,
		slash:     e,
	}

	if e.GuildID() != nil {
		c.GuildID = *e.GuildID()
	}

	for name := range data.Options {
		if v, ok := data.OptString(name); ok {
			c.Options[name] = v
			continue
		}

		if v, ok := data.OptBool(name); ok {
			c.Options[name] = fmt.Sprintf("%t", v)
			continue
		}

		if v, ok := data.OptInt(name); ok {
			c.Options[name] = fmt.Sprintf("%d", v)
		}
	}

	if sub := data.SubCommandName; sub != nil {
		c.Args = append(c.Args, *sub)
	}

	invoke(cmd, c)
}

func onAutocomplete(ctx context.Context, b *Bot, e *events.AutocompleteInteractionCreate) {
	cmd, ok := b.registry[e.Data.CommandName]

	if !ok || cmd.Autocomplete == nil {
		_ = e.AutocompleteResult([]discord.AutocompleteChoice{})
		return
	}

	var value string
	_ = json.Unmarshal(e.Data.Focused().Value, &value)

	c := &Ctx{Context: ctx, Author: e.User(), Options: map[string]string{}, bot: b}
	choices := cmd.Autocomplete(c, value)

	if choices == nil {
		choices = []discord.AutocompleteChoice{}
	}

	if err := e.AutocompleteResult(choices); err != nil {
		state.Logger.Error("Failed to answer autocomplete", zap.String("command", cmd.Name), zap.Error(err))
	}
}

// invoke runs a command through its checks, logging like poise's pre/post hooks.
func invoke(cmd *Command, c *Ctx) {
	if len(cmd.Subcommands) > 0 && len(c.Args) > 0 {
		for _, sub := range cmd.Subcommands {
			if strings.EqualFold(sub.Name, c.Args[0]) {
				c.Args = c.Args[1:]
				invoke(sub, c)
				return
			}
		}
	}

	state.Logger.Info(fmt.Sprintf("Executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))

	defer func() {
		if rec := recover(); rec != nil {
			state.Logger.Error("Command panicked", zap.String("command", cmd.Name), zap.Any("panic", rec))
			c.Fail("There was an error running this command: internal error")
		}
	}()

	if cmd.OwnerOnly && !isOwner(c.Author.ID) {
		c.Fail("Whoa there, do you have permission to do this?: You are not an owner")
		return
	}

	for _, check := range cmd.Checks {
		if err := check(c); err != nil {
			c.Fail(fmt.Sprintf("Whoa there, do you have permission to do this?: %s", err))
			return
		}
	}

	if cmd.Run == nil {
		c.Fail("This command is currently disabled")
		return
	}

	if err := cmd.Run(c); err != nil {
		c.Fail(fmt.Sprintf("There was an error running this command: %s", err))
		return
	}

	state.Logger.Info(fmt.Sprintf("Done executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))
}

func isOwner(userID snowflake.ID) bool {
	return perms.IsConfigOwner(userID.String())
}

// buildCommands renders a bot's registry into Discord's application command shape.
func (b *Bot) buildCommands() []discord.ApplicationCommandCreate {
	cmds := make([]discord.ApplicationCommandCreate, 0, len(b.ordered))

	for _, cmd := range b.ordered {
		create := discord.SlashCommandCreate{
			Name:        cmd.Name,
			Description: truncate(cmd.Description, 100),
			Options:     cmd.Options,
		}

		for _, sub := range cmd.Subcommands {
			create.Options = append(create.Options, discord.ApplicationCommandOptionSubCommand{
				Name:        sub.Name,
				Description: truncate(sub.Description, 100),
				Options:     sub.Options,
			})
		}

		cmds = append(cmds, create)
	}

	return cmds
}

// sync makes the bot's commands in Discord match exactly what it registers:
// its commands in its own guilds, none anywhere else and none globally, so a
// renamed or moved command never lingers.
func (b *Bot) sync() error {
	client := b.client()

	if client == nil {
		return fmt.Errorf("%s is not connected", b.Name)
	}

	cmds := b.buildCommands()
	guilds := map[snowflake.ID][]discord.ApplicationCommandCreate{}

	for _, id := range b.guilds() {
		guilds[id] = cmds
	}

	return cmdsync.Sync(client, cmdsync.Plan{Name: b.Name, Guilds: guilds})
}

// SyncCommands syncs both bots' commands.
func SyncCommands() error {
	ensureRegistered()

	var errs []error

	if dclient.Ready() {
		errs = append(errs, arcadiaBot.sync())
	}

	if state.Discord != nil {
		errs = append(errs, omiBot.sync())
	}

	return errors.Join(errs...)
}

func SyncArcadiaCommands() error {
	ensureRegistered()
	return arcadiaBot.sync()
}

func SyncOmiCommands() error {
	ensureRegistered()
	return omiBot.sync()
}

func truncate(s string, max int) string {
	if s == "" {
		return "No description"
	}

	if len(s) <= max {
		return s
	}

	return s[:max]
}

func (b *Bot) indexCategories() {
	seen := make(map[string]struct{}, len(b.ordered))
	b.categories = b.categories[:0]

	for _, cmd := range b.ordered {
		if _, ok := seen[cmd.Category]; ok {
			continue
		}

		seen[cmd.Category] = struct{}{}
		b.categories = append(b.categories, cmd.Category)
	}

	sort.Strings(b.categories)
}
