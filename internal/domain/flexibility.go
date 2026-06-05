// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

type Flexibility int

const (
	Exact       Flexibility = 0
	Approximate Flexibility = 30
	Flexible    Flexibility = 60
)
