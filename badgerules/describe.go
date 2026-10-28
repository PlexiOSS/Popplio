// Copyright (C) 2026 NodeByte LTD

package badgerules

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func plural(n int64, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.FormatInt(n, 10) + " " + many
}

func Describe(raw map[string]any) (string, bool) {
	if raw == nil {
		return "", false
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return "", false
	}

	var r Rule
	if err := json.Unmarshal(b, &r); err != nil || r.Type == "" {
		return "", false
	}

	switch r.Type {
	case "listed_days":
		return fmt.Sprintf("Stay approved or certified for %s.", plural(r.Value, "day", "days")), true
	case "total_votes":
		return fmt.Sprintf("Reach %s, all time.", plural(r.Value, "upvote", "upvotes")), true
	case "certified":
		return "Get the listing certified.", true
	case "premium":
		return "Have an active premium plan.", true
	case "partner":
		return "Be linked to an Omniplex partner.", true
	case "votes_cast":
		return fmt.Sprintf("Cast %s yourself. Automatic votes don't count.", plural(r.Value, "upvote", "upvotes")), true
	case "monthly_top":
		return fmt.Sprintf("Finish a calendar month in the top %d voters.", r.Value), true
	case "active_months":
		return fmt.Sprintf("Vote in %s.", plural(r.Value, "different calendar month", "different calendar months")), true
	case "owns_listings":
		return fmt.Sprintf("Own %s, directly or through a team.", plural(r.Value, "approved or certified bot or server", "approved or certified bots and servers")), true
	case "owns_certified":
		return "Own a certified bot or server, directly or through a team.", true
	case "reviews_written":
		return fmt.Sprintf("Write %s.", plural(r.Value, "review", "reviews")), true
	case "server_booster":
		return "Boost the Omniplex Discord server.", true
	case "has_role":
		return "Hold a particular role in the Omniplex Discord server.", true
	}

	return "", false
}
