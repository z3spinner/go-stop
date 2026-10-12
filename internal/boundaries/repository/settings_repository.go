// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "context"

// SettingsRepository is a generic key/value store used for runtime-provisioned
// configuration and operational flags (e.g. idempotency guards).
// *postgres.SettingsRepo satisfies this interface.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, bool, error)
	InsertIfAbsent(ctx context.Context, key, value string) error
}
