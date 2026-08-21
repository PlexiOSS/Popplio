package perms

import (
	"fmt"
	"slices"
	"strings"
)

type Perm string

func (p Perm) String() string {
	return string(p)
}

type Definition struct {
	ID          Perm     `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Dangerous   bool     `json:"dangerous,omitempty"`
	Legacy      []string `json:"-"`
}

type Catalogue struct {
	Domain string
	Super  Perm
	defs   []Definition
	index  map[Perm]Definition
}

func NewCatalogue(domain string, super Perm, defs []Definition) *Catalogue {
	c := &Catalogue{
		Domain: domain,
		Super:  super,
		defs:   defs,
		index:  make(map[Perm]Definition, len(defs)),
	}

	for _, d := range defs {
		if _, dup := c.index[d.ID]; dup {
			panic(fmt.Sprintf("perms: duplicate permission %q in the %s catalogue", d.ID, domain))
		}

		c.index[d.ID] = d
	}

	if _, ok := c.index[super]; !ok {
		panic(fmt.Sprintf("perms: super permission %q is not declared in the %s catalogue", super, domain))
	}

	return c
}

func (c *Catalogue) Definitions() []Definition {
	return slices.Clone(c.defs)
}

func (c *Catalogue) Categories() []string {
	var out []string

	for _, d := range c.defs {
		if !slices.Contains(out, d.Category) {
			out = append(out, d.Category)
		}
	}

	return out
}

func (c *Catalogue) InCategory(category string) []Definition {
	var out []Definition

	for _, d := range c.defs {
		if strings.EqualFold(d.Category, category) {
			out = append(out, d)
		}
	}

	return out
}

func (c *Catalogue) Lookup(p Perm) (Definition, bool) {
	d, ok := c.index[p]
	return d, ok
}

func (c *Catalogue) Label(p Perm) string {
	if d, ok := c.index[p]; ok {
		return d.Name
	}

	return string(p)
}

func (c *Catalogue) Validate(p Perm) error {
	if _, ok := c.index[p]; !ok {
		return fmt.Errorf("unknown %s permission %q", c.Domain, p)
	}

	return nil
}

func (c *Catalogue) ValidateStrings(list []string) error {
	var unknown []string

	for _, s := range list {
		if _, ok := c.index[Perm(s)]; !ok {
			unknown = append(unknown, s)
		}
	}

	switch len(unknown) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("unknown %s permission %q", c.Domain, unknown[0])
	default:
		return fmt.Errorf("unknown %s permissions: %s", c.Domain, strings.Join(unknown, ", "))
	}
}

func (c *Catalogue) Suggest(text string) []Definition {
	text = strings.ToLower(strings.TrimSpace(text))

	if text == "" {
		return nil
	}

	var out []Definition

	for _, d := range c.defs {
		if strings.Contains(string(d.ID), text) || strings.Contains(strings.ToLower(d.Name), text) {
			out = append(out, d)
		}
	}

	return out
}

func ParseStrings(list []string) []Perm {
	out := make([]Perm, 0, len(list))

	for _, s := range list {
		out = append(out, Perm(s))
	}

	return out
}

func Strings(list []Perm) []string {
	out := make([]string, 0, len(list))

	for _, p := range list {
		out = append(out, string(p))
	}

	return out
}
