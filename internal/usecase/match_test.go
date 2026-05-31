package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func departure(hour, min int) time.Time {
	return time.Date(2026, 6, 1, hour, min, 0, 0, time.UTC)
}

func TestWindowsOverlap_ExactMatch(t *testing.T) {
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	req := domain.Request{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	if !usecase.WindowsOverlap(ride, req) {
		t.Error("exact same time should match")
	}
}

func TestWindowsOverlap_ExactNoMatch(t *testing.T) {
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	req := domain.Request{DepartureAt: departure(10, 0), Flexibility: domain.Exact}
	if usecase.WindowsOverlap(ride, req) {
		t.Error("different exact times should not match")
	}
}

func TestWindowsOverlap_FlexibleOverlap(t *testing.T) {
	// ride 08:00-10:00, request 09:30-10:30 — overlap 09:30-10:00
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Flexible}
	req := domain.Request{DepartureAt: departure(10, 0), Flexibility: domain.Approximate}
	if !usecase.WindowsOverlap(ride, req) {
		t.Error("overlapping windows should match")
	}
}

func TestWindowsOverlap_NoOverlap(t *testing.T) {
	// ride 09:00-09:00 (exact), request 10:00-11:00 — no overlap
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	req := domain.Request{DepartureAt: departure(10, 30), Flexibility: domain.Approximate}
	if usecase.WindowsOverlap(ride, req) {
		t.Error("non-overlapping windows should not match")
	}
}

func TestWindowsOverlap_AdjacentWindows(t *testing.T) {
	// ride window ends at 09:30, request window starts at 09:30 — touching, should match
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Approximate}
	req := domain.Request{DepartureAt: departure(10, 0), Flexibility: domain.Approximate}
	if !usecase.WindowsOverlap(ride, req) {
		t.Error("touching windows (09:30 meets 09:30) should match")
	}
}
