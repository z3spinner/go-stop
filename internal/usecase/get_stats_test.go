// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetStats_ReturnsDelegatedStats(t *testing.T) {
	expectedStats := domain.Stats{
		TopRoutes: []domain.RouteStat{
			{Origin: "Saillans", Destination: "Crest", Count: 4},
		},
		TotalConfirmed: 42,
		TotalThisWeek:  4,
	}
	stats := &mockStatRepo{stats: expectedStats}

	uc := usecase.NewGetStats(stats)
	result, err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalConfirmed != 42 {
		t.Errorf("expected TotalConfirmed 42, got %d", result.TotalConfirmed)
	}
	if len(result.TopRoutes) != 1 {
		t.Errorf("expected 1 top route, got %d", len(result.TopRoutes))
	}
}
