// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// GetNotificationPreferences returns a phone's notification preferences (the
// defaults if it never changed anything).
type GetNotificationPreferences struct {
	prefs repository.NotificationPreferenceRepository
}

func NewGetNotificationPreferences(prefs repository.NotificationPreferenceRepository) *GetNotificationPreferences {
	return &GetNotificationPreferences{prefs: prefs}
}

func (uc *GetNotificationPreferences) Execute(phone string) (domain.NotificationPreferences, error) {
	if phone == "" {
		return domain.NotificationPreferences{}, ErrUnauthorized
	}
	return uc.prefs.Get(phone)
}

// SetNotificationPreferences saves a phone's notification preferences.
type SetNotificationPreferences struct {
	prefs repository.NotificationPreferenceRepository
}

func NewSetNotificationPreferences(prefs repository.NotificationPreferenceRepository) *SetNotificationPreferences {
	return &SetNotificationPreferences{prefs: prefs}
}

func (uc *SetNotificationPreferences) Execute(phone string, p domain.NotificationPreferences) (domain.NotificationPreferences, error) {
	if phone == "" {
		return domain.NotificationPreferences{}, ErrUnauthorized
	}
	if err := uc.prefs.SetMorningDigest(phone, p.MorningDigest); err != nil {
		return domain.NotificationPreferences{}, err
	}
	return p, nil
}
