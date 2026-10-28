package patch_user_profile

import (
	"encoding/json"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update User Profile",
		Description: "Updates a users profile. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.ProfileUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	var profile types.ProfileUpdate

	hresp, ok := uapi.MarshalReq(r, &profile)

	if !ok {
		return hresp
	}

	err := validators.ValidateExtraLinks(profile.ExtraLinks)

	if err != nil {
		return resp.BadRequest("Failed to validate extra links: " + err.Error())
	}

	if len(profile.About) > 1000 {
		return resp.BadRequest("About me is over 1000 characters!")
	}

	if state.ContainsSuspiciousMarkup(profile.About) {
		return resp.BadRequest("About me contains markup that isn't allowed (scripts, event handlers, or similar)")
	}

	if profile.PinnedListing != nil {
		if err := state.Validator.Struct(profile.PinnedListing); err != nil {
			return resp.BadRequest("Invalid pinned listing")
		}

		if profile.PinnedListing.TargetID != "" {
			if profile.PinnedListing.TargetType == "" {
				return resp.BadRequest("Pinned listing needs a target_type of bot or server")
			}

			entityPerms, err := teams.GetEntityPerms(d.Context, id, profile.PinnedListing.TargetType, profile.PinnedListing.TargetID)

			if err != nil || entityPerms.IsEmpty() {
				return resp.BadRequest("You can only pin a bot or server you own or manage")
			}
		}
	}

	extraLinksJSON, err := json.Marshal(profile.ExtraLinks)

	if err != nil {
		return resp.Err("Error marshaling extra links", err, zap.String("userID", d.Auth.ID))
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	err = q.UpdateUsersUpdatedAt(d.Context, id)

	if err != nil {
		return resp.Err("Error while updating updated_at", err, zap.String("userID", d.Auth.ID))
	}

	// Update extra links
	err = q.UpdateUserExtraLinks(d.Context, db.UpdateUserExtraLinksParams{
		ExtraLinks: extraLinksJSON,
		UserID:     id,
	})

	if err != nil {
		return resp.Err("Error while updating extra links", err, zap.String("userID", d.Auth.ID))
	}

	err = q.UpdateUserAbout(d.Context, db.UpdateUserAboutParams{
		About:  pgtype.Text{String: profile.About, Valid: profile.About != ""},
		UserID: id,
	})

	if err != nil {
		return resp.Err("Error while updating about", err, zap.String("userID", d.Auth.ID))
	}

	if profile.PinnedListing != nil {
		params := db.SetUserPinnedListingParams{UserID: id}

		if profile.PinnedListing.TargetID != "" {
			params.TargetType = pgtype.Text{String: profile.PinnedListing.TargetType, Valid: true}
			params.TargetID = pgtype.Text{String: profile.PinnedListing.TargetID, Valid: true}
		}

		if err := q.SetUserPinnedListing(d.Context, params); err != nil {
			return resp.Err("Error while updating pinned listing", err, zap.String("userID", d.Auth.ID))
		}
	}

	if profile.CaptchaSponsorEnabled != nil {
		err = q.UpdateUserCaptchaSponsorEnabled(d.Context, db.UpdateUserCaptchaSponsorEnabledParams{
			CaptchaSponsorEnabled: *profile.CaptchaSponsorEnabled,
			UserID:                id,
		})

		if err != nil {
			return resp.Err("Error while updating captcha sponsor enabled", err, zap.String("userID", d.Auth.ID))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
