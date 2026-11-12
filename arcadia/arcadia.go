package arcadia

import (
	"context"
	"popplio/botstatus"
	"popplio/brandemoji"
	"sync"
	"time"

	"popplio/arcadia/bot"
	"popplio/arcadia/dclient"
	"popplio/arcadia/panel"
	"popplio/arcadia/tasks"
	"popplio/state"

	discordbot "github.com/disgoorg/disgo/bot"
	"go.uber.org/zap"
)

type Arcadia struct {
	panel  *panel.Server
	cancel context.CancelFunc
	tasks  *sync.WaitGroup
}

func Start(parent context.Context) *Arcadia {
	ctx, cancel := context.WithCancel(parent)

	a := &Arcadia{
		panel:  panel.New(),
		cancel: cancel,
	}

	if err := dclient.Setup(ctx, bot.Listener(ctx)); err != nil {
		state.Logger.Error("Staff bot failed to start; the panel will run without Discord", zap.Error(err))
	} else {
		brandemoji.LoadAsync(brandemoji.Arcadia, dclient.Get(), "arcadia")
		botstatus.Run(ctx, "arcadia", func() *discordbot.Client { return dclient.Get() }, botstatus.Arcadia)
	}

	if state.Discord != nil {
		state.Discord.AddEventListeners(bot.OmiListener(ctx))
		go syncOmiWhenReady(ctx)
		botstatus.Run(ctx, "omi", func() *discordbot.Client { return state.Discord }, botstatus.Omi)
	}

	go func() {
		if err := a.panel.Start(ctx); err != nil {
			state.Logger.Error("Panel server stopped with an error", zap.Error(err))
		}
	}()

	a.tasks = tasks.Start(ctx)

	return a
}

func (a *Arcadia) Stop(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.panel.Shutdown(ctx); err != nil {
		state.Logger.Error("Panel server shutdown failed", zap.Error(err))
	}

	a.cancel()

	defer dclient.Close(ctx)

	if a.tasks != nil {
		done := make(chan struct{})

		go func() {
			a.tasks.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			state.Logger.Warn("Timed out waiting for arcadia tasks to stop")
		}
	}
}

func syncOmiWhenReady(ctx context.Context) {
	deadline := time.Now().Add(3 * time.Minute)

	for {
		if _, ok := state.Discord.Caches.Guild(state.Config.Servers.Main); ok {
			break
		}

		if time.Now().After(deadline) {
			state.Logger.Warn("Omi never saw the main server, syncing its commands anyway")
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}

	if err := bot.SyncOmiCommands(); err != nil {
		state.Logger.Error("Failed to sync Omi's application commands", zap.Error(err))
	}
}
