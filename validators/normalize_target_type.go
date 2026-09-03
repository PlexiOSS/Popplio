// Copyright (C) 2026 NodeByte LTD

package validators

import "strings"

func NormalizeTargetType(targetType string) string {
	switch targetType {
	// Bot
	case "bots":
		return "bot"
	// User
	case "users":
		return "user"
	case "user":
		return "user"
	// Server
	case "servers":
		return "server"
	case "server":
		return "server"
	// Teams
	case "teams":
		return "team"
	case "team":
		return "team"
	// Packs
	case "packs":
		return "pack"
	case "pack":
		return "pack"
	default:
		return strings.TrimSuffix(targetType, "s")
	}
}
