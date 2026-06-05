// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "github.com/z3spinner/go-stop/internal/domain"

type InterestRepository interface {
	Save(interest domain.Interest) error
	FindByID(id string) (domain.Interest, error)
	FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error)
	FindByRide(rideID string) ([]domain.Interest, error)
	Accept(id string) error
	Delete(id string) error
	// CountByRides returns a map of ride ID → interest count for the given ride IDs.
	CountByRides(rideIDs []string) (map[string]int, error)
}
