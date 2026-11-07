// Copyright (C) 2026 NodeByte LTD

package types

import "time"

type BoosterPerkTarget struct {
	TargetType string `json:"target_type" description:"bot or server"`
	TargetID   string `json:"target_id" description:"The listing's ID"`
	Name       string `json:"name" description:"The listing's name, or empty if it can't be looked up"`
	Avatar     string `json:"avatar" description:"The listing's avatar URL, or empty"`
}

type BoosterPerks struct {
	Boosting      bool               `json:"boosting" description:"Whether the user is boosting the main Omniplex Discord server right now"`
	InServer      bool               `json:"in_server" description:"Whether the user is in the main Omniplex Discord server"`
	BoostingSince *time.Time         `json:"boosting_since" description:"When Omniplex first saw the current boost"`
	Target        *BoosterPerkTarget `json:"target" description:"The listing the monthly premium goes to, or null if none is picked"`
	LastClaimAt   *time.Time         `json:"last_claim_at" description:"When booster premium was last applied"`
	NextRenewalAt *time.Time         `json:"next_renewal_at" description:"When the next month of premium is due. Null without a listing or a boost"`
	LastError     *string            `json:"last_error" description:"Why the last renewal failed, if it did"`
	JustApplied   bool               `json:"just_applied" description:"Whether this request applied a month of premium"`
	PremiumDays   int                `json:"premium_days" description:"Days of premium each renewal adds"`
	AutoVoteSlots int                `json:"auto_vote_slots" description:"Extra Auto-vote slots boosters get"`
	RenewalDays   int                `json:"renewal_days" description:"Days between renewals"`
}

type BoosterPerksUpdate struct {
	TargetType *string `json:"target_type" validate:"omitempty,oneof=bot server" msg:"target_type must be bot or server"`
	TargetID   *string `json:"target_id" validate:"omitempty,numeric" msg:"target_id must be a bot or server ID"`
}

type AutoVotePricing struct {
	PriceCents           int `json:"price_cents" description:"The monthly plan's price in US cents"`
	YearlyPriceCents     int `json:"yearly_price_cents" description:"The yearly plan's price in US cents"`
	SlotPriceCents       int `json:"slot_price_cents" description:"One extra slot on the monthly plan, per month"`
	YearlySlotPriceCents int `json:"yearly_slot_price_cents" description:"One extra slot on the yearly plan, per year"`
	IncludedSlots        int `json:"included_slots" description:"Listings included with a subscription"`
	MaxExtraSlots        int `json:"max_extra_slots" description:"The most extra slots a subscription can have"`
}
