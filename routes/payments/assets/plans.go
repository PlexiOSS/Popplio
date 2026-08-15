// Package assets holds the payment logic shared between endpoints.
//
// It defines the purchasable plans, resolves a payment into the perks it
// grants, and determines a user's server-booster status — all of which both
// the Stripe and PayPal flows need.
package assets

import (
	"popplio/types"
	"time"
)

var Plans = []types.PaymentPlan{
	{
		ID:         "bronze",
		Name:       "Bronze Plan",
		Benefit:    "1 month of premium",
		TimePeriod: 24 * 30,
		Price:      1.99,
	},
	{
		ID:         "silver",
		Name:       "Silver Plan",
		Benefit:    "6 months of premium",
		TimePeriod: 24 * 30 * 6,
		Price:      4.99,
	},
	{
		ID:      "gold",
		Name:    "Gold Plan",
		Benefit: "1 year of premium",
		TimePeriod: func() int {
			currentYear := time.Now().Year()
			days := 365
			if currentYear%4 == 0 && (currentYear%100 != 0 || currentYear%400 != 0) {
				days = 366
			}

			// TimePeriod is in hours (see GivePerks' make_interval(hours => ...)),
			// not days — this used to hand out ~365 hours (~15 days) of premium
			// instead of a year.
			return days * 24
		}(),
		Price: 7.99,
	},
}
