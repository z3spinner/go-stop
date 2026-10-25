// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

// AlertExpiryRepository tracks the expiry of open-ended alerts (anytime or
// daily requests): which ones are about to expire, whether their owner has been
// warned, and renewing them. Implemented by the request repository.
type AlertExpiryRepository interface {
	// FindExpiringBefore returns open-ended alerts that expire before `before`
	// and whose current expiry hasn't been warned about yet.
	FindExpiringBefore(before time.Time) ([]domain.Request, error)
	// MarkExpiryWarned records that the alert's expiry at expiresAt was warned about.
	MarkExpiryWarned(id string, expiresAt time.Time) error
	// SetExpiry moves the alert's expiry (renewal).
	SetExpiry(id string, expiresAt time.Time) error
}
