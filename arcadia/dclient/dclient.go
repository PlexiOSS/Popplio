package dclient

import (
	"context"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"go.uber.org/zap"
)

var client bot.Client

func Get() bot.Client {
	if client == nil {
		panic("arcadia: Discord client used before dclient.Setup")
	}
	return client
}

func Ready() bool {
	return client != nil
}

func Setup(ctx context.Context, listeners ...bot.EventListener) error {
	token := state.Config.Arcadia.Token

	if token == "" {
		return errors.New("arcadia.token is empty: the staff bot needs its own Discord token")
	}

	opts := []bot.ConfigOpt{
		bot.WithRestClientConfigOpts(state.ProxyRestOpts(token)...),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(intents()),
			gateway.WithCompress(true),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds | cache.FlagMembers | cache.FlagPresences | cache.FlagRoles),
		),
		bot.WithEventListeners(listeners...),
	}

	c, err := disgo.New(token, opts...)

	if err != nil {
		return fmt.Errorf("failed to build the staff bot client: %w", err)
	}

	if err := c.OpenGateway(ctx); err != nil {
		return fmt.Errorf("failed to connect the staff bot to the gateway: %w", err)
	}

	client = c

	if err := c.SetPresence(ctx, gateway.WithCustomActivity("Watching the review queue")); err != nil {
		state.Logger.Error("Failed to set staff bot presence", zap.Error(err))
	}

	state.Logger.Info("Staff bot connected", zap.String("applicationID", c.ApplicationID().String()))

	return nil
}

func intents() gateway.Intents {
	i := gateway.IntentGuilds |
		gateway.IntentGuildMembers |
		gateway.IntentGuildPresences |
		gateway.IntentGuildModeration

	if state.Config.Arcadia.PrefixCommands {
		i |= gateway.IntentGuildMessages | gateway.IntentMessageContent
	}

	return i
}

func Close(ctx context.Context) {
	if client == nil {
		return
	}

	client.Close(ctx)
	client = nil
}
