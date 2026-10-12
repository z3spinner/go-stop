// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package notification

import "github.com/z3spinner/go-stop/internal/domain"

type Notifier interface {
	Send(subscription domain.Subscription, message domain.Message) error
}

// Recorder is optionally implemented by a Notifier that also keeps a persistent
// copy of each message for its recipient (the in-app inbox). The usecase
// fan-out records once per recipient before pushing to their devices, so a
// message is kept even when the recipient has no push subscription.
type Recorder interface {
	Record(phone string, message domain.Message)
}
