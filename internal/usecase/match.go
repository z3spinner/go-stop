// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

// WindowsOverlap reports whether a ride and request have overlapping departure windows.
func WindowsOverlap(ride domain.Ride, req domain.Request) bool {
	rideEarliest := ride.DepartureAt.Add(-time.Duration(ride.Flexibility) * time.Minute)
	rideLatest := ride.DepartureAt.Add(time.Duration(ride.Flexibility) * time.Minute)
	reqEarliest := req.DepartureAt.Add(-time.Duration(req.Flexibility) * time.Minute)
	reqLatest := req.DepartureAt.Add(time.Duration(req.Flexibility) * time.Minute)
	return !rideLatest.Before(reqEarliest) && !rideEarliest.After(reqLatest)
}
