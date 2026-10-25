// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "github.com/z3spinner/go-stop/internal/domain"

// NotificationPreferenceRepository stores per-phone notification preferences.
// A phone that never changed anything has domain.DefaultNotificationPreferences.
type NotificationPreferenceRepository interface {
	Get(phone string) (domain.NotificationPreferences, error)
	SetMorningDigest(phone string, on bool) error
	// MorningDigestOptOuts lists the phones that turned the morning digest off.
	MorningDigestOptOuts() ([]string, error)
}
