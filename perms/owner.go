package perms

import (
	"math"

	"popplio/state"
)

const OwnerRank int32 = math.MinInt32

func IsConfigOwner(userID string) bool {
	if state.Config == nil {
		return false
	}

	for _, owner := range state.Config.Arcadia.Owners {
		if owner.String() == userID {
			return true
		}
	}

	return false
}
