// Copyright (C) 2026 NodeByte LTD

package bot

func registerCommands() {
	register(
		cmdUpdate(),
		cmdDelete(),
		cmdLeaderboard(),
		cmdStats(),
	)
}
