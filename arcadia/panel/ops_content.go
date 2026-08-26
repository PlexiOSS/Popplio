package panel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/changeloggen"
	"popplio/perms"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// parsePartner is the shared validator for Create and Update.
func parsePartner(ctx context.Context, partner types.CreatePartner) error {
	var typeID string

	err := state.Pool.QueryRow(ctx, "SELECT id FROM partner_types WHERE id = $1", partner.Type).Scan(&typeID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("Partner type does not exist")
		}

		return err
	}

	// Upstream also required the partner avatar to already exist on the CDN and
	// checked its size here. That validation went with the CDN; see
	// CONFORMANCE.md.
	if len(partner.Links) == 0 {
		return errors.New("Links cannot be empty")
	}

	for _, link := range partner.Links {
		if link.Name == "" {
			return errors.New("Link name cannot be empty")
		}

		if link.Value == "" {
			return errors.New("Link URL cannot be empty")
		}

		if !strings.HasPrefix(link.Value, "https://") {
			return errors.New("Link URL must start with https://")
		}
	}

	var userID string

	err = state.Pool.QueryRow(ctx, "SELECT user_id FROM users WHERE user_id = $1", partner.UserID).Scan(&userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("User does not exist")
		}

		return err
	}

	return nil
}

type partnerRow struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Short     string    `db:"short"`
	Links     []byte    `db:"links"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
	UserID    string    `db:"user_id"`
	BotID     *string   `db:"bot_id"`
}

type partnerTypeRow struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Short     string    `db:"short"`
	Icon      string    `db:"icon"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Server) updatePartners(ctx context.Context, q *types.QUpdatePartners) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx, "SELECT id, name, short, links, type, created_at, user_id, bot_id FROM partners")

		if err != nil {
			return response{}, newError(err)
		}

		partnerRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[partnerRow])

		if err != nil {
			return response{}, newError(err)
		}

		partners := make([]types.Partner, 0, len(partnerRows))

		for _, p := range partnerRows {
			var links []types.Link

			if err := json.Unmarshal(p.Links, &links); err != nil {
				return response{}, newError(err)
			}

			partners = append(partners, types.Partner{
				ID:        p.ID,
				Name:      p.Name,
				Short:     p.Short,
				Links:     types.NonNilLinks(links),
				Type:      p.Type,
				CreatedAt: types.NewTimestamp(p.CreatedAt),
				UserID:    p.UserID,
				BotID:     p.BotID,
			})
		}

		rows, err = state.Pool.Query(ctx, "SELECT id, name, short, icon, created_at FROM partner_types")

		if err != nil {
			return response{}, newError(err)
		}

		typeRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[partnerTypeRow])

		if err != nil {
			return response{}, newError(err)
		}

		partnerTypes := make([]types.PartnerType, 0, len(typeRows))

		for _, t := range typeRows {
			partnerTypes = append(partnerTypes, types.PartnerType{
				ID:        t.ID,
				Name:      t.Name,
				Short:     t.Short,
				Icon:      t.Icon,
				CreatedAt: types.NewTimestamp(t.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, types.Partners{Partners: partners, PartnerTypes: partnerTypes}), nil
	case q.Action.Create != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to create partners [manage_partners]"), nil
		}

		partner := q.Action.Create.Partner

		exists, err := partnerExists(ctx, partner.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if exists {
			return writeText(http.StatusBadRequest, "Partner already exists"), nil
		}

		if err := parsePartner(ctx, partner); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		links, err := json.Marshal(partner.Links)

		if err != nil {
			return response{}, newError(err)
		}

		_, err = state.Pool.Exec(ctx,
			"INSERT INTO partners (id, name, short, links, type, user_id, bot_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			partner.ID, partner.Name, partner.Short, links, partner.Type, partner.UserID, partner.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Update != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to update partners [manage_partners]"), nil
		}

		partner := q.Action.Update.Partner

		exists, err := partnerExists(ctx, partner.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Partner does not already exist"), nil
		}

		if err := parsePartner(ctx, partner); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		links, err := json.Marshal(partner.Links)

		if err != nil {
			return response{}, newError(err)
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE partners SET name = $2, short = $3, links = $4, type = $5, user_id = $6, bot_id = $7 WHERE id = $1",
			partner.ID, partner.Name, partner.Short, links, partner.Type, partner.UserID, partner.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to delete partners [manage_partners]"), nil
		}

		id := q.Action.Delete.ID

		exists, err := partnerExists(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Partner does not exist"), nil
		}

		// Upstream also deleted the partner's CDN image here. That went with the
		// CDN; see CONFORMANCE.md.
		if _, err := state.Pool.Exec(ctx, "DELETE FROM partners WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No partner action was specified")
	}
}

func partnerExists(ctx context.Context, id string) (bool, error) {
	var found string

	err := state.Pool.QueryRow(ctx, "SELECT id FROM partners WHERE id = $1", id).Scan(&found)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

type changelogRow struct {
	Itag             pgtype.UUID `db:"itag"`
	Project          string      `db:"project"`
	Version          string      `db:"version"`
	Added            []string    `db:"added"`
	Updated          []string    `db:"updated"`
	Fixed            []string    `db:"fixed"`
	Removed          []string    `db:"removed"`
	ExtraDescription string      `db:"extra_description"`
	Prerelease       bool        `db:"prerelease"`
	Published        bool        `db:"published"`
	CreatedBy        string      `db:"created_by"`
	CreatedAt        time.Time   `db:"created_at"`
}

// validChangelogProject mirrors changelogs' own CHECK (project IN (...))
// constraint, checked here too so a bad value 400s with a clear message
// instead of a raw Postgres constraint-violation error.
func validChangelogProject(project string) bool {
	return project == "popplio" || project == "omniplex" || project == "keel"
}

func (s *Server) updateChangelog(ctx context.Context, q *types.QUpdateChangelog) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListEntries != nil:
		// No permission check: this is the staff panel's own listing,
		// reachable only with a valid staff session already (see the
		// authorize call above), same as blog's listing.
		rows, err := state.Pool.Query(ctx,
			"SELECT itag, project, version, added, updated, fixed, removed, extra_description, prerelease, published, created_by, created_at FROM changelogs ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		changelogRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[changelogRow])

		if err != nil {
			return response{}, newError(err)
		}

		entries := make([]types.ChangelogEntry, 0, len(changelogRows))

		for _, row := range changelogRows {
			entries = append(entries, types.ChangelogEntry{
				Itag:             impls.UUIDString(row.Itag),
				Project:          row.Project,
				Version:          row.Version,
				Added:            types.NonNilStrings(row.Added),
				Updated:          types.NonNilStrings(row.Updated),
				Fixed:            types.NonNilStrings(row.Fixed),
				Removed:          types.NonNilStrings(row.Removed),
				ExtraDescription: row.ExtraDescription,
				Prerelease:       row.Prerelease,
				Published:        row.Published,
				CreatedBy:        row.CreatedBy,
				CreatedAt:        types.NewTimestamp(row.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, entries), nil
	case q.Action.CreateEntry != nil:
		if !userPerms.Has(perms.StaffManageChangelog) {
			return writeText(http.StatusForbidden, "You do not have permission to create changelog entries [manage_changelog]"), nil
		}

		entry := q.Action.CreateEntry

		if !validChangelogProject(entry.Project) {
			return writeText(http.StatusBadRequest, "project must be 'popplio', 'omniplex', or 'keel'"), nil
		}

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO changelogs (project, version, added, updated, fixed, removed, extra_description, prerelease, published, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11, NOW()))",
			entry.Project, entry.Version, entry.Added, entry.Updated, entry.Fixed, entry.Removed, entry.ExtraDescription, entry.Prerelease, entry.Published, authData.UserID, entry.CreatedAt)

		if err != nil {
			return response{}, newError(err)
		}

		announceChangelogEntry(*entry)

		return writeNoContent(), nil
	case q.Action.UpdateEntry != nil:
		if !userPerms.Has(perms.StaffManageChangelog) {
			return writeText(http.StatusForbidden, "You do not have permission to update changelog entries [manage_changelog]"), nil
		}

		entry := q.Action.UpdateEntry

		if !validChangelogProject(entry.Project) {
			return writeText(http.StatusBadRequest, "project must be 'popplio', 'omniplex', or 'keel'"), nil
		}

		itag, err := uuid.Parse(entry.Itag)

		if err != nil {
			return response{}, newError(err)
		}

		wasPublished, exists, err := changelogPublished(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry does not exist"), nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE changelogs SET project = $2, version = $3, added = $4, updated = $5, fixed = $6, removed = $7, extra_description = $8, prerelease = $9, published = $10, created_at = COALESCE($11, created_at) WHERE itag = $1",
			itag, entry.Project, entry.Version, entry.Added, entry.Updated, entry.Fixed, entry.Removed, entry.ExtraDescription, entry.Prerelease, entry.Published, entry.CreatedAt)

		if err != nil {
			return response{}, newError(err)
		}

		// Only announce the draft -> published transition, not every edit
		// of an already-published entry (that would re-post on every typo
		// fix).
		if entry.Published && !wasPublished {
			announceChangelogEntry(types.ChangelogCreateEntry{
				Project:          entry.Project,
				Version:          entry.Version,
				Added:            entry.Added,
				Updated:          entry.Updated,
				Fixed:            entry.Fixed,
				Removed:          entry.Removed,
				ExtraDescription: entry.ExtraDescription,
				Prerelease:       entry.Prerelease,
				Published:        entry.Published,
			})
		}

		return writeNoContent(), nil
	case q.Action.DeleteEntry != nil:
		if !userPerms.Has(perms.StaffManageChangelog) {
			return writeText(http.StatusForbidden, "You do not have permission to delete changelog entries [manage_changelog]"), nil
		}

		itag, err := uuid.Parse(q.Action.DeleteEntry.Itag)

		if err != nil {
			return response{}, newError(err)
		}

		exists, err := changelogExists(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry does not exist"), nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM changelogs WHERE itag = $1", itag); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Generate != nil:
		if !userPerms.Has(perms.StaffManageChangelog) {
			return writeText(http.StatusForbidden, "You do not have permission to create changelog entries [manage_changelog]"), nil
		}

		gen := q.Action.Generate

		if !validChangelogProject(gen.Project) {
			return writeText(http.StatusBadRequest, "project must be 'popplio', 'omniplex', or 'keel'"), nil
		}

		owner, repoName, ok := changeloggen.RepoFor(gen.Project)

		if !ok {
			return writeText(http.StatusBadRequest, "no GitHub repo is configured for this project"), nil
		}

		cmp, err := changeloggen.Compare(ctx, owner, repoName, gen.Base, gen.Head)

		if err != nil {
			return writeText(http.StatusBadRequest, "Failed to compare refs on GitHub: "+err.Error()), nil
		}

		draft, err := changeloggen.Summarize(ctx, cmp)

		if err != nil {
			return response{}, newError(err)
		}

		return writeJSON(http.StatusOK, types.ChangelogDraft{
			Added:            draft.Added,
			Updated:          draft.Updated,
			Fixed:            draft.Fixed,
			Removed:          draft.Removed,
			ExtraDescription: draft.ExtraDescription,
		}), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No changelog action was specified")
	}
}

func changelogExists(ctx context.Context, itag uuid.UUID) (bool, error) {
	var count int64

	if err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM changelogs WHERE itag = $1", itag).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// changelogPublished returns the entry's current published state, and
// whether it exists at all -- used by UpdateEntry to detect a draft ->
// published transition before the update overwrites it.
func changelogPublished(ctx context.Context, itag uuid.UUID) (published bool, exists bool, err error) {
	err = state.Pool.QueryRow(ctx, "SELECT published FROM changelogs WHERE itag = $1", itag).Scan(&published)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}

	if err != nil {
		return false, false, err
	}

	return published, true, nil
}

type blogRow struct {
	Itag        pgtype.UUID `db:"itag"`
	Slug        string      `db:"slug"`
	Title       string      `db:"title"`
	Description string      `db:"description"`
	UserID      string      `db:"user_id"`
	Content     string      `db:"content"`
	CreatedAt   time.Time   `db:"created_at"`
	Draft       bool        `db:"draft"`
	Tags        []string    `db:"tags"`
}

func (s *Server) updateBlog(ctx context.Context, q *types.QUpdateBlog) (response, error) {
	// Upstream calls check_auth twice here with a TODO admitting it is wasteful;
	// once is enough.
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListEntries != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx,
			"SELECT itag, slug, title, description, user_id, content, created_at, draft, tags FROM blogs ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		blogRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[blogRow])

		if err != nil {
			return response{}, newError(err)
		}

		entries := make([]types.BlogPost, 0, len(blogRows))

		for _, row := range blogRows {
			entries = append(entries, types.BlogPost{
				Itag:        impls.UUIDString(row.Itag),
				Slug:        row.Slug,
				Title:       row.Title,
				Description: row.Description,
				UserID:      row.UserID,
				Tags:        types.NonNilStrings(row.Tags),
				Content:     row.Content,
				CreatedAt:   types.NewTimestamp(row.CreatedAt),
				Draft:       row.Draft,
			})
		}

		return writeJSON(http.StatusOK, entries), nil
	case q.Action.CreateEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to create blog entries [manage_blog]"), nil
		}

		entry := q.Action.CreateEntry

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO blogs (slug, title, description, content, tags, user_id) VALUES ($1, $2, $3, $4, $5, $6)",
			entry.Slug, entry.Title, entry.Description, entry.Content, entry.Tags, authData.UserID)

		if err != nil {
			return response{}, newError(err)
		}

		announceBlogPost(*entry)

		return writeNoContent(), nil
	case q.Action.UpdateEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to update blog entries [manage_blog]"), nil
		}

		entry := q.Action.UpdateEntry

		itag, err := uuid.Parse(entry.Itag)

		if err != nil {
			return response{}, newError(err)
		}

		wasDraft, exists, err := blogDraft(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry does not exist"), nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE blogs SET slug = $2, title = $3, description = $4, content = $5, tags = $6, draft = $7 WHERE itag = $1",
			itag, entry.Slug, entry.Title, entry.Description, entry.Content, entry.Tags, entry.Draft)

		if err != nil {
			return response{}, newError(err)
		}

		// Only announce the draft -> published transition, not every edit
		// of an already-published post.
		if !entry.Draft && wasDraft {
			announceBlogPost(types.BlogCreateEntry{
				Slug:        entry.Slug,
				Title:       entry.Title,
				Description: entry.Description,
				Content:     entry.Content,
				Tags:        entry.Tags,
			})
		}

		return writeNoContent(), nil
	case q.Action.DeleteEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to delete blog entries [manage_blog]"), nil
		}

		itag, err := uuid.Parse(q.Action.DeleteEntry.Itag)

		if err != nil {
			return response{}, newError(err)
		}

		exists, err := blogExists(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry with same id does not already exist"), nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM blogs WHERE itag = $1", itag); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No blog action was specified")
	}
}

func blogExists(ctx context.Context, itag uuid.UUID) (bool, error) {
	var count int64

	if err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM blogs WHERE itag = $1", itag).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// blogDraft returns the entry's current draft state, and whether it exists
// at all -- used by UpdateEntry to detect a draft -> published transition
// before the update overwrites it, same reasoning as changelogPublished.
func blogDraft(ctx context.Context, itag uuid.UUID) (draft bool, exists bool, err error) {
	err = state.Pool.QueryRow(ctx, "SELECT draft FROM blogs WHERE itag = $1", itag).Scan(&draft)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}

	if err != nil {
		return false, false, err
	}

	return draft, true, nil
}
