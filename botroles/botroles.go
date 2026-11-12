package botroles

import (
	"context"
	"errors"
	"fmt"
	"slices"

	db "popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

var ErrNotReady = errors.New("Omi has not loaded the main server yet")

type Listing struct {
	Type    string
	Partner bool
}

func (l Listing) Listed() bool {
	return l.Type == "approved" || l.Type == "certified"
}

func (l Listing) Certified() bool {
	return l.Type == "certified"
}

func Wanted(l Listing) []snowflake.ID {
	roles := []snowflake.ID{state.Config.Roles.BotRole}

	if l.Certified() && state.Config.Roles.CertBot != 0 {
		roles = append(roles, state.Config.Roles.CertBot)
	}

	if l.Partner && state.Config.Roles.PartnerBot != 0 {
		roles = append(roles, state.Config.Roles.PartnerBot)
	}

	return roles
}

func Missing(have, want []snowflake.ID) []snowflake.ID {
	var out []snowflake.ID

	for _, id := range want {
		if id != 0 && !slices.Contains(have, id) {
			out = append(out, id)
		}
	}

	return out
}

var Lookup = func(ctx context.Context, botIDs []string) (map[string]Listing, error) {
	rows, err := db.New(state.Pool).GetBotRoleStates(ctx, botIDs)

	if err != nil {
		return nil, err
	}

	out := make(map[string]Listing, len(rows))

	for _, r := range rows {
		out[r.BotID] = Listing{Type: r.BotType, Partner: r.Partner}
	}

	return out, nil
}

func Give(client *bot.Client, userID snowflake.ID, roles []snowflake.ID, reason string) error {
	var errs []error

	for _, role := range roles {
		if err := client.Rest.AddMemberRole(state.Config.Servers.Main, userID, role, rest.WithReason(reason)); err != nil {
			errs = append(errs, fmt.Errorf("role %s: %w", role, err))
		}
	}

	return errors.Join(errs...)
}

func Sync(ctx context.Context) error {
	client := state.Discord

	if client == nil {
		return nil
	}

	if _, ok := client.Caches.Guild(state.Config.Servers.Main); !ok {
		return ErrNotReady
	}

	var bots []discord.Member

	for m := range client.Caches.Members(state.Config.Servers.Main) {
		if m.User.Bot && m.User.ID != client.ApplicationID {
			bots = append(bots, m)
		}
	}

	if len(bots) == 0 {
		return nil
	}

	ids := make([]string, len(bots))

	for i, m := range bots {
		ids[i] = m.User.ID.String()
	}

	listings, err := Lookup(ctx, ids)

	if err != nil {
		return err
	}

	var given, failed int

	for _, m := range bots {
		missing := Missing(m.RoleIDs, Wanted(listings[m.User.ID.String()]))

		if len(missing) == 0 {
			continue
		}

		if err := Give(client, m.User.ID, missing, "Bot role sync"); err != nil {
			failed++
			state.Logger.Warn("botroles: failed to give a bot its roles", zap.Stringer("bot_id", m.User.ID), zap.Error(err))
			continue
		}

		given += len(missing)
	}

	if given > 0 || failed > 0 {
		state.Logger.Info("botroles: synced bot roles in the main server", zap.Int("bots", len(bots)), zap.Int("roles_given", given), zap.Int("failed", failed))
	}

	return nil
}
