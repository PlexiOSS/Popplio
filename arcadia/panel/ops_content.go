// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/changeloggen"
	"popplio/db"
	"popplio/perms"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func textFromPtr(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *v, Valid: true}
}

func ptrFromText(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}

	return &v.String
}

func parsePartner(ctx context.Context, partner types.CreatePartner) error {
	exists, err := db.New(state.Pool).CountPartnerTypeByID(ctx, partner.Type)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New("Partner type does not exist")
	}

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

	userExists, err := db.New(state.Pool).UserExists(ctx, partner.UserID)

	if err != nil {
		return err
	}

	if !userExists {
		return errors.New("User does not exist")
	}

	return nil
}

func (s *Server) updatePartners(ctx context.Context, q *types.QUpdatePartners) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	queries := db.New(state.Pool)

	switch {
	case q.Action.List != nil:
		partnerRows, err := queries.ListPartners(ctx)

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
				CreatedAt: types.NewTimestamp(p.CreatedAt.Time),
				UserID:    p.UserID,
				BotID:     ptrFromText(p.BotID),
			})
		}

		partnerTypeRows, err := queries.ListPartnerTypes(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		partnerTypes := make([]types.PartnerType, 0, len(partnerTypeRows))

		for _, t := range partnerTypeRows {
			partnerTypes = append(partnerTypes, types.PartnerType{
				ID:        t.ID,
				Name:      t.Name,
				Short:     t.Short,
				Icon:      t.Icon,
				CreatedAt: types.NewTimestamp(t.CreatedAt.Time),
			})
		}

		return writeJSON(http.StatusOK, types.Partners{Partners: partners, PartnerTypes: partnerTypes}), nil
	case q.Action.Create != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to create partners [manage_partners]"), nil
		}

		partner := q.Action.Create.Partner

		exists, err := queries.CountPartnerByID(ctx, partner.ID)

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

		err = queries.InsertPartner(ctx, db.InsertPartnerParams{
			ID:     partner.ID,
			Name:   partner.Name,
			Short:  partner.Short,
			Links:  links,
			Type:   partner.Type,
			UserID: partner.UserID,
			BotID:  textFromPtr(partner.BotID),
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Update != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to update partners [manage_partners]"), nil
		}

		partner := q.Action.Update.Partner

		exists, err := queries.CountPartnerByID(ctx, partner.ID)

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

		err = queries.UpdatePartner(ctx, db.UpdatePartnerParams{
			ID:     partner.ID,
			Name:   partner.Name,
			Short:  partner.Short,
			Links:  links,
			Type:   partner.Type,
			UserID: partner.UserID,
			BotID:  textFromPtr(partner.BotID),
		})

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to delete partners [manage_partners]"), nil
		}

		id := q.Action.Delete.ID

		exists, err := queries.CountPartnerByID(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Partner does not exist"), nil
		}

		if err := queries.DeletePartner(ctx, id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No partner action was specified")
	}
}

func validChangelogProject(project string) bool {
	return project == "popplio" || project == "omniplex" || project == "keel"
}

func (s *Server) updateChangelog(ctx context.Context, q *types.QUpdateChangelog) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	queries := db.New(state.Pool)

	switch {
	case q.Action.ListEntries != nil:
		changelogRows, err := queries.ListChangelogEntries(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		entries := make([]types.ChangelogEntry, 0, len(changelogRows))

		for _, row := range changelogRows {
			entries = append(entries, types.ChangelogEntry{
				Itag:             row.Itag,
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
				CreatedAt:        types.NewTimestamp(row.CreatedAt.Time),
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

		var createdAt pgtype.Timestamptz

		if entry.CreatedAt != nil {
			createdAt = pgtype.Timestamptz{Time: *entry.CreatedAt, Valid: true}
		}

		// added/updated/fixed/removed are all NOT NULL columns; a client that
		// omits one of these keys leaves the Go slice nil, which pgx encodes
		// as SQL NULL rather than an empty array.
		err := queries.InsertChangelogEntry(ctx, db.InsertChangelogEntryParams{
			Project:          entry.Project,
			Version:          entry.Version,
			Added:            types.NonNilStrings(entry.Added),
			Updated:          types.NonNilStrings(entry.Updated),
			Fixed:            types.NonNilStrings(entry.Fixed),
			Removed:          types.NonNilStrings(entry.Removed),
			ExtraDescription: entry.ExtraDescription,
			Prerelease:       entry.Prerelease,
			Published:        entry.Published,
			CreatedBy:        authData.UserID,
			CreatedAt:        createdAt,
		})

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

		if _, err := uuid.Parse(entry.Itag); err != nil {
			return response{}, newError(err)
		}

		wasPublished, err := queries.GetChangelogPublishedByItag(ctx, entry.Itag)

		if err != nil {
			exists, existsErr := queries.CountChangelogByItag(ctx, entry.Itag)

			if existsErr != nil {
				return response{}, newError(existsErr)
			}

			if !exists {
				return writeText(http.StatusBadRequest, "Entry does not exist"), nil
			}

			return response{}, newError(err)
		}

		var createdAt pgtype.Timestamptz

		if entry.CreatedAt != nil {
			createdAt = pgtype.Timestamptz{Time: *entry.CreatedAt, Valid: true}
		}

		err = queries.UpdateChangelogEntry(ctx, db.UpdateChangelogEntryParams{
			Itag:             entry.Itag,
			Project:          entry.Project,
			Version:          entry.Version,
			Added:            types.NonNilStrings(entry.Added),
			Updated:          types.NonNilStrings(entry.Updated),
			Fixed:            types.NonNilStrings(entry.Fixed),
			Removed:          types.NonNilStrings(entry.Removed),
			ExtraDescription: entry.ExtraDescription,
			Prerelease:       entry.Prerelease,
			Published:        entry.Published,
			CreatedAt:        createdAt,
		})

		if err != nil {
			return response{}, newError(err)
		}

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

		itag := q.Action.DeleteEntry.Itag

		if _, err := uuid.Parse(itag); err != nil {
			return response{}, newError(err)
		}

		exists, err := queries.CountChangelogByItag(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry does not exist"), nil
		}

		if err := queries.DeleteChangelogByItag(ctx, itag); err != nil {
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

func (s *Server) updateBlog(ctx context.Context, q *types.QUpdateBlog) (response, error) {

	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	queries := db.New(state.Pool)

	switch {
	case q.Action.ListEntries != nil:
		blogRows, err := queries.ListBlogEntriesFull(ctx)

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
				CreatedAt:   types.NewTimestamp(row.CreatedAt.Time),
				Draft:       row.Draft,
			})
		}

		return writeJSON(http.StatusOK, entries), nil
	case q.Action.CreateEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to create blog entries [manage_blog]"), nil
		}

		entry := q.Action.CreateEntry

		err := queries.InsertBlogEntry(ctx, db.InsertBlogEntryParams{
			Slug:        entry.Slug,
			Title:       entry.Title,
			Description: entry.Description,
			Content:     entry.Content,
			Tags:        types.NonNilStrings(entry.Tags),
			UserID:      authData.UserID,
		})

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

		itagUUID := pgtype.UUID{Bytes: itag, Valid: true}

		wasDraft, err := queries.GetBlogDraftByItag(ctx, itagUUID)

		if err != nil {
			exists, existsErr := queries.CountBlogByItag(ctx, itagUUID)

			if existsErr != nil {
				return response{}, newError(existsErr)
			}

			if !exists {
				return writeText(http.StatusBadRequest, "Entry does not exist"), nil
			}

			return response{}, newError(err)
		}

		err = queries.UpdateBlogEntry(ctx, db.UpdateBlogEntryParams{
			Itag:        itagUUID,
			Slug:        entry.Slug,
			Title:       entry.Title,
			Description: entry.Description,
			Content:     entry.Content,
			Tags:        types.NonNilStrings(entry.Tags),
			Draft:       entry.Draft,
		})

		if err != nil {
			return response{}, newError(err)
		}

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

		itagUUID := pgtype.UUID{Bytes: itag, Valid: true}

		exists, err := queries.CountBlogByItag(ctx, itagUUID)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry with same id does not already exist"), nil
		}

		if err := queries.DeleteBlogByItag(ctx, itagUUID); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No blog action was specified")
	}
}
