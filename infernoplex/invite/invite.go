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

// Same pattern as routes/servers/assets.ResolveInvite — matches an optional
// discord.gg/discord.com/invite/... prefix and captures the trailing code.
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

// ResolveInvite validates that rawInvite (whatever a server owner pasted
// into the setup wizard, or sent to Sorbet's ResolveInvite query) is a real,
// permanent invite for guildID.
//
// This deliberately never makes an HTTP request to rawInvite itself — an
// earlier version did (a GET to the raw user-supplied URL, following
// redirects, then checking the *final* URL's host after the request had
// already fired), which is a server-side request forgery: an attacker could
// point Popplio's server at an arbitrary internal/external URL and have it
// make a real request before any validation ran. Popplio's main invite
// resolution (routes/servers/assets.ResolveInvite) never had this problem
// because it only ever extracts an invite *code* from the input and hands
// that to Discord's own REST client — the only network request is always to
// Discord's API, never to attacker-controlled data. Same pattern here.
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

		if length < 30*24*time.Hour {
			return errors.New("Invite expiry must be after at least 30 days long")
		}

		return errors.New("Invite must be permanent")
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
