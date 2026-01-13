package postgres

import "testing"

func TestAppointmentTargetCount(t *testing.T) {
	if got := appointmentTargetCount(0, 10); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := appointmentTargetCount(50, 10); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
}

func TestNormalizeAppointmentWindow(t *testing.T) {
	if got := normalizeAppointmentWindow(0); got != 10 {
		t.Fatalf("expected default 10, got %d", got)
	}
	if got := normalizeAppointmentWindow(5); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
}
