package catalog

import (
	"testing"
	"time"
)

func TestValidatePublishedAtRejectsFutureTime(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	if err := validatePublishedAt(now, now.Add(time.Minute)); err == nil {
		t.Fatal("future publication time must be rejected")
	}
	if err := validatePublishedAt(now, now); err != nil {
		t.Fatalf("current publication time should be accepted: %v", err)
	}
	if err := validatePublishedAt(now, now.Add(-time.Hour)); err != nil {
		t.Fatalf("historical publication time should be accepted: %v", err)
	}
}
