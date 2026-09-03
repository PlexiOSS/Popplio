// Copyright (C) 2026 NodeByte LTD

package invite

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"popplio/db"
	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
)

var inviteCodeRegex = regexp.MustCompile(`(?:discord(?:\.gg|\.com/invite|app\.com/invite)/)?([a-zA-Z0-9-]+)/?$`)

type ErrorKind string

const (
	ErrGeneric                          ErrorKind = "Generic"
	ErrServerNotFound                   ErrorKind = "ServerNotFound"
	ErrServerNeedsLoginForInvite        ErrorKind = "ServerNeedsLoginForInvite"
	ErrUserIsBlacklisted                ErrorKind = "UserIsBlacklisted"
	ErrServerHasNoInvite                ErrorKind = "ServerHasNoInvite"
	ErrServerHasInvalidInvite           ErrorKind = "ServerHasInvalidInvite"
	ErrServerTypeNotApprovedOrCertified ErrorKind = "ServerTypeNotApprovedOrCertified"
	ErrServerStateNotPublic             ErrorKind = "ServerStateNotPublic"
)

type CreateInviteError struct {
	Kind    ErrorKind
	Message string
}

func (e *CreateInviteError) Error() string {
	switch e.Kind {
	case ErrGeneric:
		return e.Message
	case ErrServerNotFound:
		return "Server not found"
	case ErrServerNeedsLoginForInvite:
		return "In order to view this server, you must login!"
	case ErrUserIsBlacklisted:
		return "User is blacklisted from this server"
	case ErrServerHasNoInvite:
		return "Server has no invite"
	case ErrServerHasInvalidInvite:
		return "Server has an invalid invite"
	case ErrServerTypeNotApprovedOrCertified:
		return "Server is not approved or certified"
	case ErrServerStateNotPublic:
		return "Server is not public/unlisted and hence invites to it cannot be created unless explicitly whitelisted"
	default:
		return string(e.Kind)
	}
}

func generic(format string, args ...any) *CreateInviteError {
	return &CreateInviteError{Kind: ErrGeneric, Message: fmt.Sprintf(format, args...)}
}

func ResolveInvite(ctx context.Context, guildID snowflake.ID, rawInvite string) error {
	match := inviteCodeRegex.FindStringSubmatch(strings.TrimSpace(rawInvite))

	if len(match) < 2 || match[1] == "" {
		return errors.New("Invalid invite URL: No code could be parsed")
	}

	code := match[1]

	inv, err := dclient.Get().Rest().GetInvite(code)

	if err != nil {
		return fmt.Errorf("Failed to fetch invite: %w", err)
	}

	if inv.Guild == nil {
		return errors.New("Could not fetch information about this guild")
	}

	if inv.Guild.ID != guildID {
		return errors.New("This invite does not correspond to this server")
	}

	if inv.ExpiresAt != nil {
		length := time.Until(*inv.ExpiresAt)

		return fmt.Errorf("Invite must be permanent (expires in %s)", length.Round(time.Hour))
	}

	return nil
}

func CreateInviteForUser(ctx context.Context, guildID snowflake.ID, userID *string, skipChecks bool) (string, *CreateInviteError) {
	row, err := db.New(state.Pool).GetServerInviteEligibility(ctx, guildID.String())

	if errors.Is(err, pgx.ErrNoRows) {
		return "", &CreateInviteError{Kind: ErrServerNotFound}
	}

	if err != nil {
		return "", generic("Failed to fetch server data: %v", err)
	}

	loginRequired, blacklistedUsers, inviteStr, serverType, serverState := row.LoginRequiredForInvite, row.BlacklistedUsers, row.Invite, row.Type, row.State

	if !skipChecks {
		if loginRequired {
			if userID == nil {
				return "", &CreateInviteError{Kind: ErrServerNeedsLoginForInvite}
			}

			if slices.Contains(blacklistedUsers, *userID) {
				return "", &CreateInviteError{Kind: ErrUserIsBlacklisted}
			}
		}

		if serverType != "approved" && serverType != "certified" {
			return "", &CreateInviteError{Kind: ErrServerTypeNotApprovedOrCertified}
		}

		if serverState != "public" && serverState != "unlisted" {
			return "", &CreateInviteError{Kind: ErrServerStateNotPublic}
		}
	}

	if inviteStr == "none" {
		return "", &CreateInviteError{Kind: ErrServerHasNoInvite}
	}

	parts := strings.Split(inviteStr, ":")

	if len(parts) < 2 {
		return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
	}

	switch parts[0] {
	case "invite_url":
		return strings.Join(parts[1:], ":"), nil

	case "per_user":
		channelID, err := snowflake.Parse(parts[1])

		if err != nil {
			return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
		}

		maxUses := 1

		if len(parts) >= 3 {
			n, err := strconv.Atoi(parts[2])

			if err != nil {
				return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
			}

			maxUses = n
		}

		maxAge := 300

		if len(parts) >= 4 {
			n, err := strconv.Atoi(parts[3])

			if err != nil {
				return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
			}

			maxAge = n
		}

		reason := "Invite created for anonymous user"

		if userID != nil {
			reason = fmt.Sprintf("Invite created for user %s", *userID)
		}

		created, err := dclient.Get().Rest().CreateInvite(channelID, discord.InviteCreate{
			MaxAge:  &maxAge,
			MaxUses: &maxUses,
			Unique:  true,
		}, rest.WithReason(reason))

		if err != nil {
			return "", generic("Failed to create invite: %v", err)
		}

		return created.URL(), nil

	default:
		return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
	}
}
