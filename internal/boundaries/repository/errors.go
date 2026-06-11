// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "errors"

// ErrDuplicateRide is returned when a write would collide with the ride dedup
// key (uq_rides_dedup) — e.g. editing a ride so it matches another of the same
// driver's rides. Callers map this to a 409 Conflict.
var ErrDuplicateRide = errors.New("duplicate ride")
