// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetActiveRequests_ReturnsAllActive(t *testing.T) {
	reqs := &mockRequestRepo{matching: []domain.Request{
		{ID: "r1", Origin: "Saillans", Destination: "Crest"},
		{ID: "r2", Origin: "Crest", Destination: "Die"},
	}}

	out, err := usecase.NewGetActiveRequests(reqs).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 requests, got %d", len(out))
	}
}

func TestGetActiveRequests_ReturnsEmptySliceNotNil(t *testing.T) {
	reqs := &mockRequestRepo{} // FindAllActive returns nil

	out, err := usecase.NewGetActiveRequests(reqs).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Error("expected a non-nil empty slice (so JSON encodes [] not null)")
	}
	if len(out) != 0 {
		t.Errorf("expected 0 requests, got %d", len(out))
	}
}
