// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package notification

import "github.com/z3spinner/go-stop/internal/domain"

type Notifier interface {
	Send(subscription domain.Subscription, message domain.Message) error
}
