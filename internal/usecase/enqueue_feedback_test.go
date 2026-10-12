// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestEnqueueFeedback_CallsRepoWith24hBound(t *testing.T) {
	q := &mockFeedbackQueue{}
	uc := usecase.NewEnqueueFeedback(q)

	before := time.Now()
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if !q.enqueueCalled {
		t.Fatal("expected EnqueueStartedRides to be called")
	}
	wantLo := before.Add(-usecase.FeedbackEnqueueBound)
	wantHi := after.Add(-usecase.FeedbackEnqueueBound)
	if q.enqueueWindowArg.Before(wantLo) || q.enqueueWindowArg.After(wantHi) {
		t.Errorf("windowStartAfter %v not within [%v, %v]", q.enqueueWindowArg, wantLo, wantHi)
	}
}
