// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

type Subscription struct {
	ID       string
	Phone    string
	Endpoint string
	Keys     PushKeys
}

type PushKeys struct {
	P256DH string
	Auth   string
}
