package bot

func registerCommands() {
	register(
		cmdUpdate(),
		cmdDelete(),
		cmdLeaderboard(),
		cmdStats(),
	)
}
