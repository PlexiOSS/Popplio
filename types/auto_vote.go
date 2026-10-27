package types

import "time"

type AutoVoteTargetInput struct {
	TargetType  string `json:"target_type" validate:"required,oneof=bot server" msg:"target_type must be bot or server"`
	TargetID    string `json:"target_id" validate:"required,numeric" msg:"target_id must be a bot or server ID"`
	Days        []int  `json:"days" validate:"max=7,unique,dive,min=0,max=6" msg:"days must be unique weekday numbers from 0 (Sunday) to 6 (Saturday)"`
	StartMinute int    `json:"start_minute" validate:"min=0,max=1439" msg:"start_minute must be between 0 and 1439"`
	EndMinute   int    `json:"end_minute" validate:"min=1,max=1440,gtfield=StartMinute" msg:"end_minute must be after start_minute and at most 1440"`
	Timezone    string `json:"timezone" validate:"required,max=64" msg:"timezone must be an IANA timezone like Europe/London"`
}

type AutoVoteTargetsUpdate struct {
	Targets []AutoVoteTargetInput `json:"targets" validate:"max=3,dive" msg:"You can auto-vote for up to 3 listings"`
}

type AutoVoteTarget struct {
	TargetType     string     `json:"target_type" description:"bot or server"`
	TargetID       string     `json:"target_id" description:"The listing's ID"`
	Name           string     `json:"name" description:"The listing's name"`
	Avatar         string     `json:"avatar" description:"The listing's avatar URL"`
	Days           []int      `json:"days" description:"Weekdays to vote on, 0 (Sunday) to 6 (Saturday). Empty means every day"`
	StartMinute    int        `json:"start_minute" description:"Start of the daily voting window, in minutes after midnight in the timezone"`
	EndMinute      int        `json:"end_minute" description:"End of the daily voting window (exclusive), in minutes after midnight in the timezone"`
	Timezone       string     `json:"timezone" description:"IANA timezone the schedule uses"`
	LastVotedAt    *time.Time `json:"last_voted_at" description:"When an automatic vote was last cast"`
	LastSkipReason *string    `json:"last_skip_reason" description:"Why the last automatic vote was skipped, if it was"`
}

type AutoVoteStatus struct {
	Subscribed        bool             `json:"subscribed" description:"Whether the user has an active auto-vote subscription"`
	Status            string           `json:"status" description:"The Stripe subscription status, or empty if the user never subscribed"`
	CurrentPeriodEnd  *time.Time       `json:"current_period_end" description:"When the current billing period ends"`
	CancelAtPeriodEnd bool             `json:"cancel_at_period_end" description:"Whether the subscription ends at the end of the current period"`
	PriceCents        int              `json:"price_cents" description:"The monthly price in US cents"`
	MaxTargets        int              `json:"max_targets" description:"How many listings can be auto-voted for"`
	Targets           []AutoVoteTarget `json:"targets" description:"The listings being auto-voted for"`
}
