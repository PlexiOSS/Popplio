package types

import "time"

type AutoVoteTargetInput struct {
	TargetType  string `json:"target_type" validate:"required,oneof=bot server" msg:"target_type must be bot or server"`
	TargetID    string `json:"target_id" validate:"required,numeric" msg:"target_id must be a bot or server ID"`
	Days        []int  `json:"days" validate:"max=7,unique,dive,min=0,max=6" msg:"days must be unique weekday numbers from 0 (Sunday) to 6 (Saturday)"`
	StartMinute int    `json:"start_minute" validate:"min=0,max=1439" msg:"start_minute must be between 0 and 1439"`
	EndMinute   int    `json:"end_minute" validate:"min=1,max=1440,gtfield=StartMinute" msg:"end_minute must be after start_minute and at most 1440"`
	Timezone    string `json:"timezone" validate:"required,max=64" msg:"timezone must be an IANA timezone like Europe/London"`
	Paused      bool   `json:"paused"`
}

type AutoVoteTargetsUpdate struct {
	Targets []AutoVoteTargetInput `json:"targets" validate:"max=10,dive" msg:"You can auto-vote for up to 10 listings"`
	Notify  *string               `json:"notify" validate:"omitempty,oneof=off each daily" msg:"notify must be off, each or daily"`
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
	CooldownUntil  *time.Time `json:"cooldown_until" description:"When the user's vote cooldown on this listing ends, whether their last vote was automatic or by hand. Null if they aren't on cooldown"`
	InSchedule     bool       `json:"in_schedule" description:"Whether the listing's schedule allows a vote right now"`
	Paused         bool       `json:"paused" description:"Whether automatic votes for this listing are paused"`
}

type AutoVoteStatus struct {
	Subscribed        bool             `json:"subscribed" description:"Whether the user has an active auto-vote subscription"`
	Status            string           `json:"status" description:"The Stripe subscription status, or empty if the user never subscribed"`
	CurrentPeriodEnd  *time.Time       `json:"current_period_end" description:"When the current billing period ends"`
	CancelAtPeriodEnd bool             `json:"cancel_at_period_end" description:"Whether the subscription ends at the end of the current period"`
	PriceCents        int              `json:"price_cents" description:"The monthly price in US cents"`
	YearlyPriceCents  int              `json:"yearly_price_cents" description:"The yearly plan's price in US cents"`
	MaxTargets        int              `json:"max_targets" description:"How many listings can be auto-voted for, including extra slots"`
	ExtraSlots        int              `json:"extra_slots" description:"Extra listing slots bought on top of the included ones"`
	MaxExtraSlots     int              `json:"max_extra_slots" description:"The most extra slots a subscription can have"`
	SlotPriceCents    int              `json:"slot_price_cents" description:"The price of one extra slot in US cents, per billing interval"`
	BillingInterval   string           `json:"billing_interval" description:"month or year. month if the user never subscribed"`
	BoosterSlots      int              `json:"booster_slots" description:"Free slots the user gets for boosting the Omniplex Discord server, included in max_targets"`
	Targets           []AutoVoteTarget `json:"targets" description:"The listings being auto-voted for"`
	Notify            string           `json:"notify" description:"How the user hears about automatic votes: off, each (a notification for every vote) or daily (one summary a day)"`
}

type AutoVoteSlotsUpdate struct {
	ExtraSlots int `json:"extra_slots" validate:"min=0,max=7" msg:"extra_slots must be between 0 and 7"`
}
