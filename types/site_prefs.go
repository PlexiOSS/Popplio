// Copyright (C) 2026 NodeByte LTD

package types

import "time"

type SitePrefs struct {
	Prefs     map[string]any `json:"prefs" description:"The website's customization settings (theme, fonts, layout and content filters). Omniplex stores them as given"`
	UpdatedAt *time.Time     `json:"updated_at" description:"When they were last saved, or null if never"`
}

type SitePrefsUpdate struct {
	Prefs map[string]any `json:"prefs" validate:"required" msg:"prefs must be an object"`
}
