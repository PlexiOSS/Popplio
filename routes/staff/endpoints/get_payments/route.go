package get_payments

import (
	"net/http"
	"strconv"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Get Premium Payments",
		Description: "Lists recent premium purchases. Needs `view_payments`.",
		Params: []docs.Parameter{
			{Name: "user_id", Description: "Only purchases by this user, or for this bot or server", In: "query", Schema: docs.IdSchema},
			{Name: "revoked", Description: "true for only refunded or disputed purchases", In: "query", Schema: docs.BoolSchema},
			{Name: "limit", Description: "1 to 200. Default 100", In: "query", Schema: docs.IdSchema},
		},
		Resp: types.StaffPaymentList{},
	}
}

func optInt(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}

	i := int(v.Int32)
	return &i
}

func optText(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}

	return &v.String
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	staffPerms, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error while loading staff permissions", err)
	}

	if !staffPerms.Has(perms.StaffViewPayments) {
		return resp.Forbidden("You do not have permission to view payments [view_payments]")
	}

	params := db.ListRecentPaymentFulfillmentsParams{
		RevokedOnly: r.URL.Query().Get("revoked") == "true",
		Lim:         100,
	}

	if user := strings.TrimSpace(r.URL.Query().Get("user_id")); user != "" {
		params.UserID = pgtype.Text{String: user, Valid: true}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		n, err := strconv.Atoi(limit)

		if err != nil || n < 1 || n > 200 {
			return resp.BadRequest("limit must be between 1 and 200")
		}

		params.Lim = int32(n)
	}

	rows, err := db.New(state.Pool).ListRecentPaymentFulfillments(d.Context, params)

	if err != nil {
		return resp.Err("Failed to load payments", err)
	}

	out := types.StaffPaymentList{Payments: make([]types.StaffPayment, 0, len(rows))}

	for _, row := range rows {
		p := types.StaffPayment{
			Provider:            row.Provider,
			ExternalID:          row.ExternalID,
			UserID:              row.UserID,
			ProductID:           row.ProductID,
			ForType:             row.ForType,
			ForID:               row.ForID,
			FulfilledAt:         row.FulfilledAt.Time,
			RevokeReason:        optText(row.RevokeReason),
			PremiumHours:        optInt(row.PremiumHours),
			CreditsUsed:         optInt(row.CreditsUsed),
			CreditDiscountCents: optInt(row.CreditDiscountCents),
			CreditState:         optText(row.CreditState),
			PromoCode:           optText(row.PromoCode),
		}

		if row.RevokedAt.Valid {
			at := row.RevokedAt.Time
			p.RevokedAt = &at
		}

		out.Payments = append(out.Payments, p)
	}

	return uapi.HttpResponse{Json: out}
}
