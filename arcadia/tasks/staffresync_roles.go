package tasks

import (
	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

func modifyCorrespondingRoles(posByID map[string]cachedPosition, user snowflake.ID, removeIDs, addIDs []string) error {
	remove := collectCorrespondingRoles(posByID, removeIDs)
	add := collectCorrespondingRoles(posByID, addIDs)

	for guildID, roles := range remove {
		if !guildMemberPresent(guildID, user) {
			continue
		}

		for _, roleID := range roles {
			if err := impls.RemoveRole(guildID, user, roleID, "Removing corresponding role"); err != nil {
				return err
			}
		}
	}

	for guildID, roles := range add {
		if !guildMemberPresent(guildID, user) {
			continue
		}

		for _, roleID := range roles {
			if err := impls.AddRole(guildID, user, roleID, "Adding corresponding role"); err != nil {
				return err
			}
		}
	}

	return nil
}

func collectCorrespondingRoles(posByID map[string]cachedPosition, positionIDs []string) map[snowflake.ID][]snowflake.ID {
	out := make(map[snowflake.ID][]snowflake.ID)

	for _, id := range positionIDs {
		pos, ok := posByID[id]

		if !ok {
			continue
		}

		for _, link := range pos.CorrespondingRoles {
			var guildID snowflake.ID

			switch link.Name {
			case "main":
				guildID = state.Config.Servers.Main
			case "staff":
				guildID = state.Config.Servers.Staff
			case "testing":
				guildID = state.Config.Servers.Testing
			default:
				state.Logger.Warn("Unknown corresponding server", zap.String("name", link.Name))
				continue
			}

			roleID, err := snowflake.Parse(link.Value)

			if err != nil {
				state.Logger.Warn("Unparseable corresponding role id", zap.String("value", link.Value))
				continue
			}

			out[guildID] = append(out[guildID], roleID)
		}
	}

	return out
}

func guildMemberPresent(guildID, user snowflake.ID) bool {
	if _, ok := dclient.Get().Caches().Guild(guildID); !ok {
		state.Logger.Warn("Failed to get guild", zap.String("guildID", guildID.String()))
		return false
	}

	if !impls.MemberOnGuild(guildID, user) {
		state.Logger.Warn("User not found in server", zap.String("userID", user.String()))
		return false
	}

	return true
}
