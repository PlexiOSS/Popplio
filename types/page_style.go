// Copyright (C) 2026 NodeByte LTD

package types

import "time"

type PageStyle struct {
	Gradient      *string `json:"gradient" description:"A preset gradient ID, or null"`
	Accent        *string `json:"accent" description:"The page's accent colour as #rrggbb, or null for the viewer's own accent"`
	ImageVersion  *string `json:"image_version" description:"The approved background image's version. The image is at {CDN}/backgrounds/{bots|servers}/{id}/{version}.webp"`
	Dim           int     `json:"dim" description:"How much the background is darkened or lightened toward the page colour, 30 to 90 percent"`
	Blur          int     `json:"blur" description:"Background blur in pixels, 0 to 24"`
	ImagePosition string  `json:"image_position" description:"top, center or bottom"`
	Fixed         bool    `json:"fixed" description:"Whether the background stays put while the page scrolls"`
}

type ManagedPageStyle struct {
	PageStyle
	Premium        bool       `json:"premium" description:"Whether page styles are available: the listing has premium or an active Page Style pass. Page styles only show while they are"`
	UnlockedUntil  *time.Time `json:"unlocked_until" description:"When the listing's Page Style pass from the Credit shop runs out, if one is active"`
	PendingVersion *string    `json:"pending_version" description:"An uploaded background waiting for staff review"`
	PendingSince   *time.Time `json:"pending_since" description:"When the pending background was uploaded"`
	RejectedReason *string    `json:"rejected_reason" description:"Why staff rejected the last uploaded background"`
}

type PageStyleUpdate struct {
	Gradient      *string `json:"gradient" validate:"omitempty,max=32" msg:"gradient must be a preset ID"`
	Accent        *string `json:"accent" validate:"omitempty,hexcolor,len=7" msg:"accent must be a colour like #6366f1"`
	Dim           int     `json:"dim" validate:"min=30,max=90" msg:"dim must be between 30 and 90"`
	Blur          int     `json:"blur" validate:"min=0,max=24" msg:"blur must be between 0 and 24"`
	ImagePosition string  `json:"image_position" validate:"oneof=top center bottom" msg:"image_position must be top, center or bottom"`
	Fixed         bool    `json:"fixed"`
	RemoveImage   bool    `json:"remove_image" description:"Remove the background image, including one waiting for review"`
}

type PageImageSubmit struct {
	Version string `json:"version" validate:"required,len=16,hexadecimal" msg:"version must be 16 hex characters"`
}
