package bot

import (
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/rpc"
	"popplio/arcadia/types"
	"popplio/perms"
)

func registerCommands() {
	register(
		cmdRegister(),
		cmdHelp(),
		cmdExplainMe(),
		cmdStaff(),
		cmdInviteDB(),
		cmdInvite(),
		cmdStaffGuide(),
		cmdQueue(),
		cmdClaim(),
		cmdUnclaim(),
		cmdApprove(),
		cmdDeny(),
		cmdAnalytics(),
		cmdInfo(),
		cmdLeaderboard(),
		cmdRefresh(),
		cmdGetBotRoles(),
		cmdRPC(),
		cmdRPCList(),
	)

	registerStaffRoleCommands()
	registerModerationCommands()
	registerHelpLinkCommands()
}

func cmdRegister() *Command {
	return &Command{
		Name:        "register",
		Category:    "Owner",
		Description: "Register application commands",
		OwnerOnly:   true,
		Run: func(c *Ctx) error {
			if err := SyncCommands(); err != nil {
				return err
			}

			return c.Ok("Registered application commands.")
		},
	}
}

func runRPCWithTarget(c *Ctx, method types.RPCMethod, targetType types.TargetType) (rpc.Success, error) {
	return rpc.Execute(c.Context, method, rpc.Handle{
		UserID:     c.Author.ID.String(),
		TargetType: targetType,
	})
}

func requirePerm(c *Ctx, perm perms.Perm) error {
	sp, err := impls.GetUserPerms(c.Context, c.Author.ID.String())

	if err != nil {
		return err
	}

	if !sp.Has(perm) {
		return fmt.Errorf("You need the %s permission to use this command", perms.Staff.Label(perm))
	}

	return nil
}

func runRPC(c *Ctx, method types.RPCMethod) (rpc.Success, error) {
	return runRPCWithTarget(c, method, types.TargetTypeBot)
}
