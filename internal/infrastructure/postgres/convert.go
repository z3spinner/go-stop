// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

// ── pgtype helpers ────────────────────────────────────────────────────────────

func uuidFrom(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func uuidTo(u pgtype.UUID) string {
	t, _ := u.Value()
	if t == nil {
		return ""
	}
	return t.(string)
}

func tsFrom(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func tsTo(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func dateFrom(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func dateTo(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

// ── Domain converters ─────────────────────────────────────────────────────────

func rideFromRow(r queries.Ride) domain.Ride {
	return domain.Ride{
		ID:            uuidTo(r.ID),
		DriverName:    r.DriverName,
		Phone:         r.Phone,
		Origin:        r.Origin,
		Destination:   r.Destination,
		Date:          dateTo(r.Date),
		DepartureAt:   tsTo(r.DepartureAt),
		Flexibility:   domain.Flexibility(r.Flexibility),
		PostedAt:      tsTo(r.PostedAt),
		ExpiresAt:     tsTo(r.ExpiresAt),
		FeedbackGiven: r.FeedbackGiven,
	}
}

func ridesFromRows(rows []queries.Ride) []domain.Ride {
	out := make([]domain.Ride, len(rows))
	for i, r := range rows {
		out[i] = rideFromRow(r)
	}
	return out
}

func requestFromRow(r queries.Request) domain.Request {
	return domain.Request{
		ID:           uuidTo(r.ID),
		SearcherName: r.SearcherName,
		Phone:        r.Phone,
		Origin:       r.Origin,
		Destination:  r.Destination,
		Date:         dateTo(r.Date),
		DepartureAt:  tsTo(r.DepartureAt),
		Flexibility:  domain.Flexibility(r.Flexibility),
		PostedAt:     tsTo(r.PostedAt),
		ExpiresAt:    tsTo(r.ExpiresAt),
	}
}

func requestsFromRows(rows []queries.Request) []domain.Request {
	out := make([]domain.Request, len(rows))
	for i, r := range rows {
		out[i] = requestFromRow(r)
	}
	return out
}

func interestFromRow(r queries.Interest) domain.Interest {
	return domain.Interest{
		ID:            uuidTo(r.ID),
		RideID:        uuidTo(r.RideID),
		SearcherPhone: r.SearcherPhone,
		SearcherName:  r.SearcherName,
		Status:        r.Status,
		CreatedAt:     tsTo(r.CreatedAt),
	}
}

func interestsFromRows(rows []queries.Interest) []domain.Interest {
	out := make([]domain.Interest, len(rows))
	for i, r := range rows {
		out[i] = interestFromRow(r)
	}
	return out
}

func subscriptionFromRow(s queries.Subscription) domain.Subscription {
	return domain.Subscription{
		ID:       uuidTo(s.ID),
		Phone:    s.Phone,
		Endpoint: s.Endpoint,
		Keys: domain.PushKeys{
			P256DH: s.P256dh,
			Auth:   s.Auth,
		},
	}
}

func feedbackTaskFromRow(q queries.FeedbackQueue) domain.FeedbackTask {
	return domain.FeedbackTask{
		ID:          uuidTo(q.ID),
		RideID:      uuidTo(q.RideID),
		Phone:       q.Phone,
		Origin:      q.Origin,
		Destination: q.Destination,
		RideDate:    dateTo(q.RideDate),
		DepartureAt: tsTo(q.DepartureAt),
		SendAfter:   tsTo(q.SendAfter),
		SentCount:   int(q.SentCount),
		LastSentAt:  tsTo(q.LastSentAt),
		CreatedAt:   tsTo(q.CreatedAt),
	}
}
