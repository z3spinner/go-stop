// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import "errors"

var ErrUnauthorized = errors.New("unauthorized")
var ErrNotFound = errors.New("not found")

// ErrNotPending is returned when an action is only valid on a pending interest
// (e.g. a searcher cancelling a request the driver has already accepted).
var ErrNotPending = errors.New("interest is not pending")

var ErrNameRequired = errors.New("name is required")
