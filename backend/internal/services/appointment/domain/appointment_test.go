package domain_test

import (
	"testing"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
)

func TestStateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.AppointmentStatus
		to      domain.AppointmentStatus
		allowed bool
	}{
		{"HeldToConfirmed", domain.StatusHeld, domain.StatusConfirmed, true},
		{"HeldToPendingApproval", domain.StatusHeld, domain.StatusPendingApproval, true},
		{"HeldToCancelled", domain.StatusHeld, domain.StatusCancelled, true},
		{"HeldToCheckedIn", domain.StatusHeld, domain.StatusCheckedIn, false},
		{"PendingApprovalToConfirmed", domain.StatusPendingApproval, domain.StatusConfirmed, true},
		{"PendingApprovalToRejected", domain.StatusPendingApproval, domain.StatusRejected, true},
		{"PendingApprovalToCancelled", domain.StatusPendingApproval, domain.StatusCancelled, true},
		{"PendingApprovalToInProgress", domain.StatusPendingApproval, domain.StatusInProgress, false},
		{"ConfirmedToCheckedIn", domain.StatusConfirmed, domain.StatusCheckedIn, true},
		{"ConfirmedToCancelled", domain.StatusConfirmed, domain.StatusCancelled, true},
		{"ConfirmedToRescheduled", domain.StatusConfirmed, domain.StatusRescheduled, true},
		{"ConfirmedToNoShow", domain.StatusConfirmed, domain.StatusNoShow, true},
		{"ConfirmedToCompleted", domain.StatusConfirmed, domain.StatusCompleted, false},
		{"CheckedInToInProgress", domain.StatusCheckedIn, domain.StatusInProgress, true},
		{"CheckedInToCancelled", domain.StatusCheckedIn, domain.StatusCancelled, true},
		{"CheckedInToCompleted", domain.StatusCheckedIn, domain.StatusCompleted, false},
		{"InProgressToCompleted", domain.StatusInProgress, domain.StatusCompleted, true},
		{"InProgressToCancelled", domain.StatusInProgress, domain.StatusCancelled, false},
		{"CompletedToAny", domain.StatusCompleted, domain.StatusCancelled, false},
		{"CancelledToConfirmed", domain.StatusCancelled, domain.StatusConfirmed, false},
		{"RescheduledToConfirmed", domain.StatusRescheduled, domain.StatusConfirmed, false},
		{"IdempotentConfirmed", domain.StatusConfirmed, domain.StatusConfirmed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := domain.CanTransitionTo(tt.from, tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%v, %v) = %v; want %v", tt.from, tt.to, allowed, tt.allowed)
			}
			err := domain.ValidateTransition(tt.from, tt.to)
			if tt.allowed && err != nil {
				t.Errorf("ValidateTransition(%v, %v) unexpected error: %v", tt.from, tt.to, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("ValidateTransition(%v, %v) expected error, got nil", tt.from, tt.to)
			}
		})
	}
}

func TestAppointmentHold_IsExpired(t *testing.T) {
	now := time.Now().UTC()
	hold := domain.AppointmentHold{
		Status:    domain.HoldStatusActive,
		ExpiresAt: now.Add(10 * time.Minute),
	}

	if hold.IsExpired(now) {
		t.Errorf("expected active hold not to be expired at %v", now)
	}

	if !hold.IsExpired(now.Add(11 * time.Minute)) {
		t.Errorf("expected hold to be expired 11 minutes later")
	}

	hold.Status = domain.HoldStatusConfirmed
	if !hold.IsExpired(now) {
		t.Errorf("expected non-active hold to report as expired/unavailable")
	}
}
