package assets

import (
	"popplio/boosters"
	"popplio/types"

	"github.com/disgoorg/snowflake/v2"
)

func CheckUserBoosterStatus(id snowflake.ID) types.BoosterStatus {
	member, boosting := boosters.Check(id)

	if !member {
		return types.BoosterStatus{
			Remark:    "Member not found on server",
			IsBooster: false,
		}
	}

	if !boosting {
		return types.BoosterStatus{
			Remark:    "Member is not boosting the server",
			IsBooster: false,
		}
	}

	return types.BoosterStatus{IsBooster: true}
}
