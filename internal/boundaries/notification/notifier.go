// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package notification

import "github.com/z3spinner/go-stop/internal/domain"

// Pusher delivers a message to one push subscription (one device).
type Pusher interface {
	Send(subscription domain.Subscription, message domain.Message) error
}

// Notifier is the port use cases depend on: push delivery plus a persistent
// per-recipient copy of each message (the in-app inbox). Recording is part of
// the interface on purpose, so any wrapper around a Notifier must forward it
// and the inbox can't be switched off silently. sendToAll records once per
// recipient before pushing, so recipients without a push subscription still
// get the message in the app.
type Notifier interface {
	Pusher
	Record(phone string, message domain.Message)
}
