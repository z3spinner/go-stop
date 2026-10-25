// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

// NotificationPreferences are one phone's choices about optional notifications.
// The zero value is not the default: use DefaultNotificationPreferences.
type NotificationPreferences struct {
	MorningDigest bool // the daily summary of the day's rides and requests
}

// DefaultNotificationPreferences applies to a phone that never changed anything.
func DefaultNotificationPreferences() NotificationPreferences {
	return NotificationPreferences{MorningDigest: true}
}
