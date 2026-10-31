package promo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"

	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"github.com/stripe/stripe-go/v75/coupon"
	"github.com/stripe/stripe-go/v75/promotioncode"
	"go.uber.org/zap"
)

const (
	ProductPremium  = "premium"
	ProductAutoVote = "autovote"
	MinChargeCents  = 50
	MaxRepeatMonths = 36
)

var (
	Products = []string{ProductPremium, ProductAutoVote}
	codeRe   = regexp.MustCompile(`^[A-Z0-9_-]{3,40}$`)
)

type Rejection struct {
	Reason string
}

func (r *Rejection) Error() string {
	return r.Reason
}

func reject(reason string) error {
	return &Rejection{Reason: reason}
}

func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func productLabel(product string) string {
	if product == ProductAutoVote {
		return "Auto-vote"
	}

	return "premium"
}

type Code = db.GetPromoCodeByCodeRow

type Preview struct {
	Code           string `json:"code" description:"The normalised promo code"`
	Product        string `json:"product" description:"The product the code was checked against: premium or autovote"`
	Description    string `json:"description" description:"Staff-written description of the offer"`
	PercentOff     *int   `json:"percent_off" description:"Percentage taken off, if this is a percentage code"`
	AmountOffCents *int   `json:"amount_off_cents" description:"Fixed amount taken off in US cents, if this is a fixed amount code"`
	Duration       string `json:"duration" description:"How long the discount lasts on Auto-vote: once, repeating or forever"`
	DurationMonths *int   `json:"duration_months" description:"Number of months the discount repeats for, when duration is repeating"`
	SubtotalCents  int    `json:"subtotal_cents" description:"Price in US cents before the code, after any credits"`
	DiscountCents  int    `json:"discount_cents" description:"Amount the code takes off the first payment, in US cents"`
	TotalCents     int    `json:"total_cents" description:"First payment in US cents after the code"`
}

func optInt(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}

	i := int(v.Int32)
	return &i
}

func Discount(c *Code, subtotalCents int) int {
	if subtotalCents <= 0 {
		return 0
	}

	if c.PercentOff.Valid {
		return min(int(math.Round(float64(subtotalCents)*float64(c.PercentOff.Int32)/100)), subtotalCents)
	}

	return min(int(c.AmountOffCents.Int32), subtotalCents)
}

func Validate(ctx context.Context, userID, rawCode, product string, subtotalCents int) (*Code, *Preview, error) {
	if !slices.Contains(Products, product) {
		return nil, nil, reject("Unknown product")
	}

	code := Normalize(rawCode)

	if !codeRe.MatchString(code) {
		return nil, nil, reject("This promo code doesn't exist")
	}

	q := db.New(state.Pool)

	c, err := q.GetPromoCodeByCode(ctx, code)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, reject("This promo code doesn't exist")
	}

	if err != nil {
		return nil, nil, err
	}

	if !c.Active {
		return nil, nil, reject("This promo code is no longer active")
	}

	if c.ExpiresAt.Valid && !c.ExpiresAt.Time.After(time.Now()) {
		return nil, nil, reject("This promo code has expired")
	}

	if !slices.Contains(c.Products, product) {
		return nil, nil, reject("This promo code can't be used for " + productLabel(product))
	}

	if c.MaxRedemptions.Valid {
		used, err := q.CountPromoCodeRedemptions(ctx, c.ID)

		if err != nil {
			return nil, nil, err
		}

		if used >= c.MaxRedemptions.Int32 {
			return nil, nil, reject("This promo code has been fully redeemed")
		}
	}

	if c.OncePerUser {
		used, err := q.CountUserPromoCodeRedemptions(ctx, db.CountUserPromoCodeRedemptionsParams{PromoCodeID: c.ID, UserID: userID})

		if err != nil {
			return nil, nil, err
		}

		if used > 0 {
			return nil, nil, reject("You've already used this promo code")
		}
	}

	if c.NewCustomersOnly {
		paid, err := q.UserHasPaidBefore(ctx, userID)

		if err != nil {
			return nil, nil, err
		}

		if paid {
			return nil, nil, reject("This promo code is only for new customers")
		}
	}

	discount := Discount(&c, subtotalCents)
	total := subtotalCents - discount

	if product == ProductPremium && total > 0 && total < MinChargeCents {
		return nil, nil, reject("With this promo code the total would be under the $0.50 minimum card payment. Try using fewer credits")
	}

	return &c, &Preview{
		Code:           c.Code,
		Product:        product,
		Description:    c.Description,
		PercentOff:     optInt(c.PercentOff),
		AmountOffCents: optInt(c.AmountOffCents),
		Duration:       c.Duration,
		DurationMonths: optInt(c.DurationMonths),
		SubtotalCents:  subtotalCents,
		DiscountCents:  discount,
		TotalCents:     total,
	}, nil
}

type CreateParams struct {
	Code             string     `json:"code"`
	Description      string     `json:"description"`
	Products         []string   `json:"products"`
	PercentOff       *int       `json:"percent_off"`
	AmountOffCents   *int       `json:"amount_off_cents"`
	Duration         string     `json:"duration"`
	DurationMonths   *int       `json:"duration_months"`
	MaxRedemptions   *int       `json:"max_redemptions"`
	ExpiresAt        *time.Time `json:"expires_at"`
	OncePerUser      bool       `json:"once_per_user"`
	NewCustomersOnly bool       `json:"new_customers_only"`
}

func (p *CreateParams) normalize() error {
	p.Code = Normalize(p.Code)
	p.Description = strings.TrimSpace(p.Description)

	if !codeRe.MatchString(p.Code) {
		return reject("Codes must be 3 to 40 characters of A-Z, 0-9, _ or -")
	}

	if len(p.Description) > 200 {
		return reject("The description can be at most 200 characters")
	}

	if len(p.Products) == 0 {
		return reject("Pick at least one product")
	}

	var products []string

	for _, product := range p.Products {
		if !slices.Contains(Products, product) {
			return reject("Unknown product " + strconv.Quote(product))
		}

		if !slices.Contains(products, product) {
			products = append(products, product)
		}
	}

	p.Products = products

	if (p.PercentOff == nil) == (p.AmountOffCents == nil) {
		return reject("Set either a percentage or a fixed amount off")
	}

	if p.PercentOff != nil && (*p.PercentOff < 1 || *p.PercentOff > 100) {
		return reject("The percentage must be between 1 and 100")
	}

	if p.AmountOffCents != nil && (*p.AmountOffCents < 1 || *p.AmountOffCents > 100000) {
		return reject("The fixed amount must be between $0.01 and $1000")
	}

	if !slices.Contains(p.Products, ProductAutoVote) {
		p.Duration = string(stripe.CouponDurationOnce)
		p.DurationMonths = nil
	}

	switch p.Duration {
	case string(stripe.CouponDurationOnce), string(stripe.CouponDurationForever):
		p.DurationMonths = nil
	case string(stripe.CouponDurationRepeating):
		if p.DurationMonths == nil || *p.DurationMonths < 1 || *p.DurationMonths > MaxRepeatMonths {
			return reject("Repeating discounts need between 1 and " + strconv.Itoa(MaxRepeatMonths) + " months")
		}
	default:
		return reject("Duration must be once, repeating or forever")
	}

	if p.MaxRedemptions != nil && *p.MaxRedemptions < 1 {
		return reject("Total uses must be at least 1")
	}

	if p.ExpiresAt != nil && !p.ExpiresAt.After(time.Now().Add(time.Minute)) {
		return reject("The expiry date must be in the future")
	}

	return nil
}

func int4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

func Create(ctx context.Context, p CreateParams, createdBy string) (pgtype.UUID, error) {
	if state.Config.Meta.StripeSecretKey == "" {
		return pgtype.UUID{}, errors.New("stripe is not configured")
	}

	if err := p.normalize(); err != nil {
		return pgtype.UUID{}, err
	}

	q := db.New(state.Pool)

	if _, err := q.GetPromoCodeByCode(ctx, p.Code); err == nil {
		return pgtype.UUID{}, reject("A promo code called " + p.Code + " already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, err
	}

	metadata := map[string]string{"promo_code": p.Code, "created_by": createdBy, "products": strings.Join(p.Products, ",")}

	couponParams := &stripe.CouponParams{
		Name:     stripe.String(p.Code),
		Duration: stripe.String(p.Duration),
		Metadata: metadata,
	}

	if p.PercentOff != nil {
		couponParams.PercentOff = stripe.Float64(float64(*p.PercentOff))
	} else {
		couponParams.AmountOff = stripe.Int64(int64(*p.AmountOffCents))
		couponParams.Currency = stripe.String(string(stripe.CurrencyUSD))
	}

	if p.DurationMonths != nil {
		couponParams.DurationInMonths = stripe.Int64(int64(*p.DurationMonths))
	}

	c, err := coupon.New(couponParams)

	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("creating Stripe coupon: %w", err)
	}

	promoParams := &stripe.PromotionCodeParams{
		Coupon:   stripe.String(c.ID),
		Code:     stripe.String(p.Code),
		Metadata: metadata,
	}

	if p.MaxRedemptions != nil {
		promoParams.MaxRedemptions = stripe.Int64(int64(*p.MaxRedemptions))
	}

	if p.ExpiresAt != nil {
		promoParams.ExpiresAt = stripe.Int64(p.ExpiresAt.Unix())
	}

	pc, err := promotioncode.New(promoParams)

	if err != nil {
		deleteCoupon(c.ID)
		return pgtype.UUID{}, fmt.Errorf("creating Stripe promotion code: %w", err)
	}

	var expires pgtype.Timestamptz

	if p.ExpiresAt != nil {
		expires = pgtype.Timestamptz{Time: *p.ExpiresAt, Valid: true}
	}

	id, err := q.InsertPromoCode(ctx, db.InsertPromoCodeParams{
		Code:                  p.Code,
		Description:           p.Description,
		Products:              p.Products,
		PercentOff:            int4(p.PercentOff),
		AmountOffCents:        int4(p.AmountOffCents),
		Duration:              p.Duration,
		DurationMonths:        int4(p.DurationMonths),
		MaxRedemptions:        int4(p.MaxRedemptions),
		ExpiresAt:             expires,
		OncePerUser:           p.OncePerUser,
		NewCustomersOnly:      p.NewCustomersOnly,
		StripeCouponID:        c.ID,
		StripePromotionCodeID: pc.ID,
		CreatedBy:             createdBy,
	})

	if err != nil {
		if _, updErr := promotioncode.Update(pc.ID, &stripe.PromotionCodeParams{Active: stripe.Bool(false)}); updErr != nil {
			state.Logger.Error("Failed to deactivate orphaned Stripe promotion code", zap.Error(updErr), zap.String("promotion_code", pc.ID))
		}

		deleteCoupon(c.ID)

		return pgtype.UUID{}, fmt.Errorf("saving promo code: %w", err)
	}

	return id, nil
}

func deleteCoupon(id string) {
	if _, err := coupon.Del(id, nil); err != nil {
		state.Logger.Error("Failed to delete orphaned Stripe coupon", zap.Error(err), zap.String("coupon", id))
	}
}

func SetActive(ctx context.Context, id pgtype.UUID, active bool) error {
	q := db.New(state.Pool)

	stripeID, err := q.GetPromoCodeStripeID(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return reject("Promo code not found")
	}

	if err != nil {
		return err
	}

	if _, err := promotioncode.Update(stripeID, &stripe.PromotionCodeParams{Active: stripe.Bool(active)}); err != nil {
		var serr *stripe.Error

		if errors.As(err, &serr) && serr.HTTPStatusCode < 500 {
			return reject("Stripe refused the change: " + serr.Msg)
		}

		return fmt.Errorf("updating Stripe promotion code: %w", err)
	}

	return q.SetPromoCodeActive(ctx, db.SetPromoCodeActiveParams{ID: id, Active: active})
}

func ReleasePending(ctx context.Context, c *Code, userID string) error {
	q := db.New(state.Pool)

	sessions, err := q.ListUserPendingPromoRedemptions(ctx, db.ListUserPendingPromoRedemptionsParams{PromoCodeID: c.ID, UserID: userID})

	if err != nil {
		return err
	}

	for _, id := range sessions {
		s, err := session.Get(id, nil)

		if err != nil {
			return fmt.Errorf("checking earlier checkout %s: %w", id, err)
		}

		switch s.Status {
		case stripe.CheckoutSessionStatusComplete:
			if s.PaymentStatus == stripe.CheckoutSessionPaymentStatusUnpaid {
				return reject("You have a payment with this promo code that is still processing")
			}

			if err := q.CompletePromoCodeRedemption(ctx, id); err != nil {
				return err
			}

			if c.OncePerUser {
				return reject("You've already used this promo code")
			}

			continue
		case stripe.CheckoutSessionStatusOpen:
			if _, err := session.Expire(id, nil); err != nil {
				return fmt.Errorf("expiring earlier checkout %s: %w", id, err)
			}
		}

		if err := q.AbandonPromoCodeRedemption(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

func Record(ctx context.Context, c *Code, userID, product, sessionID string) error {
	return db.New(state.Pool).InsertPromoCodeRedemption(ctx, db.InsertPromoCodeRedemptionParams{
		PromoCodeID:     c.ID,
		UserID:          userID,
		Product:         product,
		StripeSessionID: sessionID,
	})
}

func Complete(ctx context.Context, sessionID string) {
	if err := db.New(state.Pool).CompletePromoCodeRedemption(ctx, sessionID); err != nil {
		state.Logger.Error("Failed to mark promo code redemption completed", zap.Error(err), zap.String("session", sessionID))
	}
}

func Abandon(ctx context.Context, sessionID string) {
	if err := db.New(state.Pool).AbandonPromoCodeRedemption(ctx, sessionID); err != nil {
		state.Logger.Error("Failed to mark promo code redemption abandoned", zap.Error(err), zap.String("session", sessionID))
	}
}

func SessionDiscount(c *Code) []*stripe.CheckoutSessionDiscountParams {
	return []*stripe.CheckoutSessionDiscountParams{{PromotionCode: stripe.String(c.StripePromotionCodeID)}}
}

func ErrResponse(reason string, err error) uapi.HttpResponse {
	var rej *Rejection

	if errors.As(err, &rej) {
		return resp.BadRequest(rej.Reason)
	}

	return resp.Err(reason, err)
}

func StripeErrResponse(reason string, err error) uapi.HttpResponse {
	var serr *stripe.Error

	if errors.As(err, &serr) && serr.HTTPStatusCode >= 400 && serr.HTTPStatusCode < 500 && serr.Param != "" && strings.HasPrefix(serr.Param, "discounts") {
		return resp.BadRequest("Stripe couldn't apply this promo code: " + serr.Msg)
	}

	return resp.Err(reason, err)
}
