package types

import (
	"fmt"
	"time"
)

// PartnerAction is the union of partner operations.
type PartnerAction struct {
	List   *Unit
	Create *PartnerCreate
	Update *PartnerCreate
	Delete *PartnerDelete
}

type PartnerCreate struct {
	Partner CreatePartner `json:"partner"`
}

type PartnerDelete struct {
	ID string `json:"id"`
}

func (a *PartnerAction) UnmarshalJSON(data []byte) error {
	*a = PartnerAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("PartnerAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "Create":
		a.Create = &PartnerCreate{}
		return decodeVariant("PartnerAction", name, payload, a.Create)
	case "Update":
		a.Update = &PartnerCreate{}
		return decodeVariant("PartnerAction", name, payload, a.Update)
	case "Delete":
		a.Delete = &PartnerDelete{}
		return decodeVariant("PartnerAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("PartnerAction", name)
	}

	return expectUnit("PartnerAction", name, payload)
}

func (a PartnerAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Update != nil:
		return encodeVariant("Update", a.Update)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("PartnerAction: no variant set")
	}
}

// CreatePartner is the partner payload accepted by Create and Update.
type CreatePartner struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Short  string  `json:"short"`
	BotID  *string `json:"bot_id"`
	Links  []Link  `json:"links"`
	Type   string  `json:"type"`
	UserID string  `json:"user_id"`
}

type Partner struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Short     string    `json:"short"`
	Links     []Link    `json:"links"`
	BotID     *string   `json:"bot_id"`
	Type      string    `json:"type"`
	CreatedAt Timestamp `json:"created_at"`
	UserID    string    `json:"user_id"`
}

type PartnerType struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Short     string    `json:"short"`
	Icon      string    `json:"icon"`
	CreatedAt Timestamp `json:"created_at"`
}

type Partners struct {
	Partners     []Partner     `json:"partners"`
	PartnerTypes []PartnerType `json:"partner_types"`
}

// ChangelogAction is the union of changelog operations: curated, staff-
// authored release entries for both Popplio, Omniplex, and Keel.
type ChangelogAction struct {
	ListEntries *Unit
	CreateEntry *ChangelogCreateEntry
	UpdateEntry *ChangelogUpdateEntry
	DeleteEntry *ChangelogDeleteEntry
	Generate    *ChangelogGenerateRequest
}

// ChangelogGenerateRequest asks for a not-yet-saved draft entry built from
// the merged PRs between two git refs on the project's GitHub repo.
type ChangelogGenerateRequest struct {
	Project string `json:"project"`
	Base    string `json:"base"`
	Head    string `json:"head"`
}

// ChangelogDraft is the generated result — everything a ChangelogCreateEntry
// needs except project/version/prerelease/published, which the admin sets
// by hand before saving.
type ChangelogDraft struct {
	Added            []string `json:"added"`
	Updated          []string `json:"updated"`
	Fixed            []string `json:"fixed"`
	Removed          []string `json:"removed"`
	ExtraDescription string   `json:"extra_description"`
}

// Project is either "popplio" or "omniplex" on every DTO below — validated
// against changelogs' own CHECK constraint at the SQL layer.
type ChangelogCreateEntry struct {
	Project          string   `json:"project"`
	Version          string   `json:"version"`
	ExtraDescription string   `json:"extra_description"`
	Prerelease       bool     `json:"prerelease"`
	Published        bool     `json:"published"`
	Added            []string `json:"added"`
	Updated          []string `json:"updated"`
	Fixed            []string `json:"fixed"`
	Removed          []string `json:"removed"`
	// CreatedAt lets staff backfill a release's real date instead of the
	// moment the entry happened to be drafted -- nil falls back to NOW().
	CreatedAt *time.Time `json:"created_at"`
}

type ChangelogUpdateEntry struct {
	Itag             string   `json:"itag"`
	Project          string   `json:"project"`
	Version          string   `json:"version"`
	ExtraDescription string   `json:"extra_description"`
	Prerelease       bool     `json:"prerelease"`
	Published        bool     `json:"published"`
	Added            []string `json:"added"`
	Updated          []string `json:"updated"`
	Fixed            []string `json:"fixed"`
	Removed          []string `json:"removed"`
	// CreatedAt, like on Create, is nil to leave the entry's existing date
	// untouched -- an update never resets it to "now" on its own.
	CreatedAt *time.Time `json:"created_at"`
}

type ChangelogDeleteEntry struct {
	Itag string `json:"itag"`
}

func (a *ChangelogAction) UnmarshalJSON(data []byte) error {
	*a = ChangelogAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("ChangelogAction: %w", err)
	}

	switch name {
	case "ListEntries":
		a.ListEntries = unitSet()
	case "CreateEntry":
		a.CreateEntry = &ChangelogCreateEntry{}
		return decodeVariant("ChangelogAction", name, payload, a.CreateEntry)
	case "UpdateEntry":
		a.UpdateEntry = &ChangelogUpdateEntry{}
		return decodeVariant("ChangelogAction", name, payload, a.UpdateEntry)
	case "DeleteEntry":
		a.DeleteEntry = &ChangelogDeleteEntry{}
		return decodeVariant("ChangelogAction", name, payload, a.DeleteEntry)
	case "Generate":
		a.Generate = &ChangelogGenerateRequest{}
		return decodeVariant("ChangelogAction", name, payload, a.Generate)
	default:
		return errUnknownVariant("ChangelogAction", name)
	}

	return expectUnit("ChangelogAction", name, payload)
}

func (a ChangelogAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListEntries != nil:
		return encodeUnit("ListEntries")
	case a.CreateEntry != nil:
		return encodeVariant("CreateEntry", a.CreateEntry)
	case a.UpdateEntry != nil:
		return encodeVariant("UpdateEntry", a.UpdateEntry)
	case a.DeleteEntry != nil:
		return encodeVariant("DeleteEntry", a.DeleteEntry)
	case a.Generate != nil:
		return encodeVariant("Generate", a.Generate)
	default:
		return nil, fmt.Errorf("ChangelogAction: no variant set")
	}
}

type ChangelogEntry struct {
	Itag             string    `json:"itag"`
	Project          string    `json:"project"`
	Version          string    `json:"version"`
	Added            []string  `json:"added"`
	Updated          []string  `json:"updated"`
	Fixed            []string  `json:"fixed"`
	Removed          []string  `json:"removed"`
	CreatedBy        string    `json:"created_by"`
	CreatedAt        Timestamp `json:"created_at"`
	ExtraDescription string    `json:"extra_description"`
	Prerelease       bool      `json:"prerelease"`
	Published        bool      `json:"published"`
}

// BlogAction is the union of blog operations.
type BlogAction struct {
	ListEntries *Unit
	CreateEntry *BlogCreateEntry
	UpdateEntry *BlogUpdateEntry
	DeleteEntry *BlogDeleteEntry
}

type BlogCreateEntry struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
}

type BlogUpdateEntry struct {
	Itag        string   `json:"itag"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	Draft       bool     `json:"draft"`
}

type BlogDeleteEntry struct {
	Itag string `json:"itag"`
}

func (a *BlogAction) UnmarshalJSON(data []byte) error {
	*a = BlogAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("BlogAction: %w", err)
	}

	switch name {
	case "ListEntries":
		a.ListEntries = unitSet()
	case "CreateEntry":
		a.CreateEntry = &BlogCreateEntry{}
		return decodeVariant("BlogAction", name, payload, a.CreateEntry)
	case "UpdateEntry":
		a.UpdateEntry = &BlogUpdateEntry{}
		return decodeVariant("BlogAction", name, payload, a.UpdateEntry)
	case "DeleteEntry":
		a.DeleteEntry = &BlogDeleteEntry{}
		return decodeVariant("BlogAction", name, payload, a.DeleteEntry)
	default:
		return errUnknownVariant("BlogAction", name)
	}

	return expectUnit("BlogAction", name, payload)
}

func (a BlogAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListEntries != nil:
		return encodeUnit("ListEntries")
	case a.CreateEntry != nil:
		return encodeVariant("CreateEntry", a.CreateEntry)
	case a.UpdateEntry != nil:
		return encodeVariant("UpdateEntry", a.UpdateEntry)
	case a.DeleteEntry != nil:
		return encodeVariant("DeleteEntry", a.DeleteEntry)
	default:
		return nil, fmt.Errorf("BlogAction: no variant set")
	}
}

type BlogPost struct {
	Itag        string    `json:"itag"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      string    `json:"user_id"`
	CreatedAt   Timestamp `json:"created_at"`
	Content     string    `json:"content"`
	Draft       bool      `json:"draft"`
	Tags        []string  `json:"tags"`
}
