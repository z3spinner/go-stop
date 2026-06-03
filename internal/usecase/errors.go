package usecase

import "errors"

var ErrUnauthorized = errors.New("unauthorized")

// ErrNotPending is returned when an action is only valid on a pending interest
// (e.g. a searcher cancelling a request the driver has already accepted).
var ErrNotPending = errors.New("interest is not pending")
