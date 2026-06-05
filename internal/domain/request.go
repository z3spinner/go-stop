// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

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
