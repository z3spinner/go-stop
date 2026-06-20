// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "github.com/z3spinner/go-stop/internal/domain"

type ContactOfferRepository interface {
	Save(offer domain.ContactOffer) error
	FindByRequestAndOfferer(requestID, offererPhone string) (domain.ContactOffer, error)
	ListByRequest(requestID string) ([]domain.ContactOffer, error)
}
