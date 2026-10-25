// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

// Open-ended alerts (anytime, or a daily time) have no date of their own to
// expire with. They last AlertLifetime from posting or from their last renewal;
// AlertExpiryNotice before that, their owner is asked whether to keep them.
// Without this, alerts outlived the people who posted them by months.
const (
	AlertLifetime     = 30 * 24 * time.Hour
	AlertExpiryNotice = 3 * 24 * time.Hour
)

type Request struct {
	ID           string
	SearcherName string
	Phone        string
	Origin       string
	Destination  string
	Date         time.Time   // zero → anytime (no date constraint)
	DepartureAt  time.Time   // zero → day or anytime (no time constraint)
	Flexibility  Flexibility // only meaningful when DepartureAt is set
	PostedAt     time.Time
	ExpiresAt    time.Time
}
