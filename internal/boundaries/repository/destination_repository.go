// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

type DestinationRepository interface {
	GetAll() ([]string, error)
}
