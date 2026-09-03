// Copyright (C) 2026 NodeByte LTD

package seo

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

type Sitemap struct {
	XMLName xml.Name      `xml:"urlset"`
	XMLNS   string        `xml:"xmlns,attr"`
	Urls    []*SitemapURL `xml:"url"`
}

type SitemapURL struct {
	XMLName     xml.Name `xml:"url"`
	Name        string   `xml:"name,omitempty"`
	Category    string   `xml:"category,omitempty"`
	Description string   `xml:"description,omitempty"`
	Loc         string   `xml:"loc"`
	ChangeFreq  string   `xml:"changefreq"`
	LastMod     string   `xml:"lastmod"`
	Priority    string   `xml:"priority"`
}

func (m *MapGenerator) AddToSitemap(ctx context.Context, f Fetcher, sitemap *Sitemap, category, id string, priority float64) error {
	e, err := m.Add(ctx, f, id)

	if err != nil {
		return err
	}

	sitemap.Urls = append(sitemap.Urls, &SitemapURL{
		Name:        e.Name,
		Category:    category,
		Description: e.Description,
		Loc:         e.URL,
		ChangeFreq:  "daily",
		LastMod:     e.UpdatedAt.Format(time.RFC3339),
		Priority:    fmt.Sprint(priority),
	})

	return nil
}
