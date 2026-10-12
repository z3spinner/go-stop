// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type ContactOfferRepo struct{ q *queries.Queries }

func NewContactOfferRepo(pool *pgxpool.Pool) *ContactOfferRepo {
	return &ContactOfferRepo{q: queries.New(pool)}
}

func (r *ContactOfferRepo) Save(o domain.ContactOffer) error {
	return r.q.InsertContactOffer(context.Background(), queries.InsertContactOfferParams{
		ID:           uuidFrom(o.ID),
		RequestID:    uuidFrom(o.RequestID),
		OffererPhone: o.OffererPhone,
		OffererName:  o.OffererName,
	})
}

func (r *ContactOfferRepo) FindByRequestAndOfferer(requestID, offererPhone string) (domain.ContactOffer, error) {
	row, err := r.q.GetContactOfferByRequestAndOfferer(context.Background(), queries.GetContactOfferByRequestAndOffererParams{
		RequestID:    uuidFrom(requestID),
		OffererPhone: offererPhone,
	})
	if err != nil {
		return domain.ContactOffer{}, errors.New("contact offer not found")
	}
	return contactOfferFromRow(row), nil
}

func (r *ContactOfferRepo) ListByRequest(requestID string) ([]domain.ContactOffer, error) {
	rows, err := r.q.ListContactOffersByRequest(context.Background(), uuidFrom(requestID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.ContactOffer, len(rows))
	for i, row := range rows {
		out[i] = contactOfferFromRow(row)
	}
	return out, nil
}

func contactOfferFromRow(r queries.ContactOffer) domain.ContactOffer {
	return domain.ContactOffer{
		ID:           uuidTo(r.ID),
		RequestID:    uuidTo(r.RequestID),
		OffererPhone: r.OffererPhone,
		OffererName:  r.OffererName,
		CreatedAt:    tsTo(r.CreatedAt),
	}
}
