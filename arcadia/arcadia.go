package arcadia

import (
	"context"
	"sync"
	"time"

	"popplio/arcadia/bot"
	"popplio/arcadia/dclient"
	"popplio/arcadia/panel"
	"popplio/arcadia/tasks"
	"popplio/state"

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
