// Copyright (C) 2026 NodeByte LTD

package seo

import (
	"context"
	"time"
)

type Entity struct {
	ID          string
	Type        string
	AvatarURL   string
	Name        string
	Description string
	URL         string
	Author      *Entity
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Fetcher interface {
	Type() string
	Fetch(ctx context.Context, mg *MapGenerator, id string) (*Entity, error)
}

type MapGenerator struct {
	Done map[string]map[string]*Entity
}

func (m *MapGenerator) Cache(e *Entity) {
	if m.Done == nil {
		m.Done = make(map[string]map[string]*Entity)
	}

	if m.Done[e.Type] == nil {
		m.Done[e.Type] = make(map[string]*Entity)
	}

	m.Done[e.Type][e.ID] = e
}

func (m *MapGenerator) Add(ctx context.Context, f Fetcher, id string) (*Entity, error) {
	if m.Done == nil {
		m.Done = make(map[string]map[string]*Entity)
	}

	if m.Done[f.Type()] == nil {
		m.Done[f.Type()] = make(map[string]*Entity)
	}

	if m.Done[f.Type()][id] != nil {
		return m.Done[f.Type()][id], nil
	}

	e, err := f.Fetch(ctx, m, id)

	if err != nil {
		return nil, err
	}

	m.Cache(e)

	return e, nil
}
