// Copyright (C) 2026 NodeByte LTD

package types

import "time"

type ServerChangelog struct {
	ID        string    `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	Content   string    `db:"content" json:"content"`
	Version   string    `db:"version" json:"version"`
	CreatedBy string    `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ServerChangelogList struct {
	Changelogs []ServerChangelog `json:"changelogs"`
}

type CreateServerChangelog struct {
	Title   string `json:"title" validate:"required,min=1,max=200" msg:"The title must be between 1 and 200 characters"`
	Content string `json:"content" validate:"required,min=1,max=10000" msg:"The update must be between 1 and 10,000 characters"`
	Version string `json:"version" validate:"max=50" msg:"The version can be at most 50 characters"`
}

type ServerChangelogFeedEntry struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Name      string    `json:"name" description:"The server's name"`
	Avatar    string    `json:"avatar" description:"The server's icon URL"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
