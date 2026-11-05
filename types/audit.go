package types

import "time"

type AuditEvent struct {
	ID          int64          `json:"id,string" description:"The event's ID. Newer events have larger IDs"`
	CreatedAt   time.Time      `json:"created_at" description:"When the event happened"`
	Source      string         `json:"source" description:"Where it came from: api, panel, system, stripe or paypal"`
	ActorType   string         `json:"actor_type" description:"Who did it: user, bot, server, team, staff, system or anonymous"`
	ActorID     *string        `json:"actor_id" description:"The actor's ID, if known"`
	ActorName   *string        `json:"actor_name" description:"The actor's name, when it is a known user, bot, server or team"`
	ActorAvatar *string        `json:"actor_avatar" description:"The actor's avatar URL, when known"`
	Action      string         `json:"action" description:"What happened, such as an API operation ID, panel.UpdatePromoCodes.Create or payment.refunded"`
	TargetType  *string        `json:"target_type" description:"The kind of thing acted on, if any"`
	TargetID    *string        `json:"target_id" description:"The ID of the thing acted on, if any"`
	TargetName  *string        `json:"target_name" description:"The target's name, when it is a known user, bot, server or team"`
	Status      *int           `json:"status" description:"The HTTP status of the request, for api and panel events"`
	IP          *string        `json:"ip" description:"The client IP, for panel events. Only shown to staff who can view sensitive data"`
	Data        map[string]any `json:"data" description:"Extra detail. Secrets and request bodies of API calls are never stored"`
}

type AuditEventList struct {
	Events     []AuditEvent `json:"events" description:"Newest first"`
	NextBefore *string      `json:"next_before" description:"Pass as before to get the next page. Null on the last page"`
}

type AutoVoteStaffEntry struct {
	UserID               string           `json:"user_id" description:"The subscriber's user ID"`
	Status               string           `json:"status" description:"The Stripe subscription status, or empty if the user never subscribed"`
	Active               bool             `json:"active" description:"Whether Auto-vote is running for this user"`
	CurrentPeriodEnd     *time.Time       `json:"current_period_end" description:"When the current billing period ends"`
	CancelAtPeriodEnd    bool             `json:"cancel_at_period_end" description:"Whether the subscription ends at the end of the period"`
	SubscribedAt         *time.Time       `json:"subscribed_at" description:"When the subscription was first recorded"`
	StripeSubscriptionID string           `json:"stripe_subscription_id" description:"The Stripe subscription ID"`
	StripeCustomerID     string           `json:"stripe_customer_id" description:"The Stripe customer ID"`
	AutoVotes30d         int              `json:"auto_votes_30d" description:"Automatic votes cast for this user in the last 30 days"`
	Targets              []AutoVoteTarget `json:"targets" description:"The listings the user auto-votes for, with their current status"`
}

type AutoVoteStaffOverview struct {
	Entries []AutoVoteStaffEntry `json:"entries" description:"Every user with a subscription or saved Auto-vote listings, newest subscription first"`
}

type StaffPayment struct {
	Provider            string     `json:"provider" description:"stripe or paypal"`
	ExternalID          string     `json:"external_id" description:"The Stripe checkout session or PayPal order ID"`
	UserID              string     `json:"user_id" description:"The buyer's user ID"`
	ProductID           string     `json:"product_id" description:"The product, currently always premium"`
	ForType             string     `json:"for_type" description:"bot or server"`
	ForID               string     `json:"for_id" description:"The bot or server that got premium"`
	FulfilledAt         time.Time  `json:"fulfilled_at" description:"When premium was applied"`
	PremiumHours        *int       `json:"premium_hours" description:"How many hours of premium the purchase added. Null for purchases made before this was recorded"`
	RevokedAt           *time.Time `json:"revoked_at" description:"When the purchase was refunded or disputed and premium was taken back"`
	RevokeReason        *string    `json:"revoke_reason" description:"refunded or disputed"`
	CreditsUsed         *int       `json:"credits_used" description:"Vote credits applied at checkout"`
	CreditDiscountCents *int       `json:"credit_discount_cents" description:"What those credits took off, in US cents"`
	CreditState         *string    `json:"credit_state" description:"held, spent or refunded"`
	PromoCode           *string    `json:"promo_code" description:"The promo code used, if any"`
}

type StaffPaymentList struct {
	Payments []StaffPayment `json:"payments" description:"Newest first"`
}
