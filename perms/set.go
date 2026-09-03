// Copyright (C) 2026 NodeByte LTD

package perms

import (
	"encoding/json"
	"maps"
	"slices"
)

type Set struct {
	cat  *Catalogue
	held map[Perm]struct{}
}

func (c *Catalogue) NewSet(list ...Perm) Set {
	s := Set{cat: c, held: make(map[Perm]struct{}, len(list))}

	for _, p := range list {
		s.held[p] = struct{}{}
	}

	return s
}

func (c *Catalogue) SetFromStrings(list []string) Set {
	return c.NewSet(ParseStrings(list)...)
}

type Role struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Index int32  `json:"index"`
	Perms []Perm `json:"perms"`
}

func (c *Catalogue) Resolve(roles []Role, extras ...Perm) Set {
	size := len(extras)

	for _, r := range roles {
		size += len(r.Perms)
	}

	s := Set{cat: c, held: make(map[Perm]struct{}, size)}

	for _, r := range roles {
		for _, p := range r.Perms {
			s.held[p] = struct{}{}
		}
	}

	for _, p := range extras {
		s.held[p] = struct{}{}
	}

	return s
}

func (c *Catalogue) ResolveStrings(list []string) Set {
	return c.SetFromStrings(list)
}

func (s Set) Has(p Perm) bool {
	if len(s.held) == 0 {
		return false
	}

	if s.cat != nil {
		if _, ok := s.held[s.cat.Super]; ok {
			return true
		}
	}

	_, ok := s.held[p]

	return ok
}

func (s Set) HasAll(list ...Perm) bool {
	for _, p := range list {
		if !s.Has(p) {
			return false
		}
	}

	return true
}

func (s Set) HasAny(list ...Perm) bool {
	for _, p := range list {
		if s.Has(p) {
			return true
		}
	}

	return false
}

func (s Set) IsSuper() bool {
	if s.cat == nil {
		return false
	}

	_, ok := s.held[s.cat.Super]

	return ok
}

func (s Set) Len() int {
	return len(s.held)
}

func (s Set) IsEmpty() bool {
	return len(s.held) == 0
}

func (s Set) All() []Perm {
	out := make([]Perm, 0, len(s.held))

	if s.cat != nil {
		for _, d := range s.cat.defs {
			if _, ok := s.held[d.ID]; ok {
				out = append(out, d.ID)
			}
		}
	}

	extra := make([]Perm, 0)

	for p := range s.held {
		if s.cat != nil {
			if _, declared := s.cat.index[p]; declared {
				continue
			}
		}

		extra = append(extra, p)
	}

	slices.Sort(extra)

	return append(out, extra...)
}

func (s Set) Undeclared() []Perm {
	if s.cat == nil {
		return nil
	}

	var out []Perm

	for p := range s.held {
		if _, ok := s.cat.index[p]; !ok {
			out = append(out, p)
		}
	}

	slices.Sort(out)

	return out
}

func (s Set) Strings() []string {
	return Strings(s.All())
}

func (s Set) With(list ...Perm) Set {
	out := s.Clone()

	if out.held == nil {
		out.held = make(map[Perm]struct{}, len(list))
	}

	for _, p := range list {
		out.held[p] = struct{}{}
	}

	return out
}

func (s Set) Without(list ...Perm) Set {
	out := s.Clone()

	for _, p := range list {
		delete(out.held, p)
	}

	return out
}

func (s Set) Clone() Set {
	return Set{cat: s.cat, held: maps.Clone(s.held)}
}

func (s Set) Equal(other Set) bool {
	return maps.Equal(s.held, other.held)
}

func (s Set) Diff(other Set) []Perm {
	changed := make([]Perm, 0)

	for p := range s.held {
		if _, ok := other.held[p]; !ok {
			changed = append(changed, p)
		}
	}

	for p := range other.held {
		if _, ok := s.held[p]; !ok {
			changed = append(changed, p)
		}
	}

	slices.Sort(changed)

	return changed
}

func (s Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Strings())
}
