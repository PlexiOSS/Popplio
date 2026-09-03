// Copyright (C) 2026 NodeByte LTD

package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

type Announcement struct {
	UserID       string                  `db:"author" json:"-"`
	Author       *dovetypes.PlatformUser `json:"author"`
	ID           pgtype.UUID             `db:"id" json:"id"`
	Title        string                  `db:"title" json:"title"`
	Content      string                  `db:"content" json:"content"`
	LastModified time.Time               `db:"modified_date" json:"last_modified"`
	Status       string                  `db:"status" json:"status"`
	Target       pgtype.Text             `db:"target" json:"target"`
}

type AnnouncementList struct {
	Announcements []Announcement `json:"announcements"`
}
