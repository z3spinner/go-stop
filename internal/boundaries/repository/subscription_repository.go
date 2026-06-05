// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "github.com/z3spinner/go-stop/internal/domain"

type SubscriptionRepository interface {
	Save(subscription domain.Subscription) error
	// FindByPhone returns ALL subscriptions for a phone (one per device).
	FindByPhone(phone string) ([]domain.Subscription, error)
	Delete(phone string) error
	// DeleteByEndpoint removes a specific device subscription (called on 410 Gone).
	DeleteByEndpoint(endpoint string) error
}
