// Copyright (C) 2026 NodeByte LTD

package brandemoji

import (
	"sync"

	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"go.uber.org/zap"
)

type Set struct {
	mu sync.RWMutex
	m  map[string]string
}

var (
	Popplio     = &Set{}
	Arcadia     = &Set{}
	Infernoplex = &Set{}
)

func (s *Set) Load(client bot.Client) (int, error) {
	emojis, err := client.Rest().GetApplicationEmojis(client.ApplicationID())

	if err != nil {
		return 0, err
	}

	m := make(map[string]string, len(emojis))

	for _, e := range emojis {
		m[e.Name] = e.Mention()
	}

	s.mu.Lock()
	s.m = m
	s.mu.Unlock()

	return len(m), nil
}

func (s *Set) Get(name, fallback string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.m[name]; ok {
		return v
	}

	return fallback
}

func LoadAsync(s *Set, client bot.Client, label string) {
	if client == nil {
		return
	}

	go func() {
		n, err := s.Load(client)

		if err != nil {
			state.Logger.Warn("Failed to load application emojis, using Unicode fallbacks", zap.String("bot", label), zap.Error(err))
			return
		}

		state.Logger.Info("Loaded application emojis", zap.String("bot", label), zap.Int("count", n))
	}()
}
