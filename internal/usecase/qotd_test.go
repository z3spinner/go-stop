// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"testing"
	"time"
)

func TestQuoteOfTheDay(t *testing.T) {
	day := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	titleFr, bodyFr := quoteOfTheDay("fr", day)
	if titleFr != "Citation du jour" {
		t.Errorf("fr title = %q", titleFr)
	}
	if bodyFr == "" {
		t.Fatal("empty fr body")
	}

	// Unknown locale falls back to French.
	titleXx, bodyXx := quoteOfTheDay("xx", day)
	if titleXx != "Citation du jour" || bodyXx != bodyFr {
		t.Errorf("unknown locale should fall back to fr, got title=%q body=%q", titleXx, bodyXx)
	}

	// A different language yields different text on the same day.
	if _, bodyEn := quoteOfTheDay("en", day); bodyEn == bodyFr {
		t.Error("en and fr bodies should differ")
	}

	// A different quote the next day, and the same one after a full cycle.
	if _, next := quoteOfTheDay("fr", day.AddDate(0, 0, 1)); next == bodyFr {
		t.Error("expected a different quote the next day")
	}
	if _, cycled := quoteOfTheDay("fr", day.AddDate(0, 0, len(qotdQuotes))); cycled != bodyFr {
		t.Error("expected the same quote after a full cycle")
	}
}
