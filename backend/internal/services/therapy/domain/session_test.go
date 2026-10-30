package domain_test

import (
	"testing"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
)

func TestSessionTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.SessionStatus
		to      domain.SessionStatus
		allowed bool
	}{
		{"CheckedInToInProgress", domain.SessionStatusCheckedIn, domain.SessionStatusInProgress, true},
		{"CheckedInToAborted", domain.SessionStatusCheckedIn, domain.SessionStatusAbortedClinical, true},
		{"CheckedInToCompleted", domain.SessionStatusCheckedIn, domain.SessionStatusCompleted, false},
		{"InProgressToCompleted", domain.SessionStatusInProgress, domain.SessionStatusCompleted, true},
		{"InProgressToComplication", domain.SessionStatusInProgress, domain.SessionStatusComplicationAlert, true},
		{"InProgressToAborted", domain.SessionStatusInProgress, domain.SessionStatusAbortedClinical, true},
		{"ComplicationToInProgress", domain.SessionStatusComplicationAlert, domain.SessionStatusInProgress, true},
		{"ComplicationToAborted", domain.SessionStatusComplicationAlert, domain.SessionStatusAbortedClinical, true},
		{"ComplicationToCompleted", domain.SessionStatusComplicationAlert, domain.SessionStatusCompleted, false},
		{"CompletedToAny", domain.SessionStatusCompleted, domain.SessionStatusInProgress, false},
		{"AbortedToAny", domain.SessionStatusAbortedClinical, domain.SessionStatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := domain.CanTransitionSession(tt.from, tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionSession(%v, %v) = %v; want %v", tt.from, tt.to, allowed, tt.allowed)
			}
			err := domain.ValidateSessionTransition(tt.from, tt.to)
			if tt.allowed && err != nil {
				t.Errorf("ValidateSessionTransition(%v, %v) unexpected error: %v", tt.from, tt.to, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("ValidateSessionTransition(%v, %v) expected error, got nil", tt.from, tt.to)
			}
		})
	}
}

func TestStepTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.StepStatus
		to      domain.StepStatus
		allowed bool
	}{
		{"NotStartedToInProgress", domain.StepStatusNotStarted, domain.StepStatusInProgress, true},
		{"NotStartedToSkipped", domain.StepStatusNotStarted, domain.StepStatusSkipped, true},
		{"NotStartedToCompleted", domain.StepStatusNotStarted, domain.StepStatusCompleted, false},
		{"InProgressToPaused", domain.StepStatusInProgress, domain.StepStatusPaused, true},
		{"InProgressToCompleted", domain.StepStatusInProgress, domain.StepStatusCompleted, true},
		{"PausedToInProgress", domain.StepStatusPaused, domain.StepStatusInProgress, true},
		{"PausedToCompleted", domain.StepStatusPaused, domain.StepStatusCompleted, true},
		{"CompletedToAny", domain.StepStatusCompleted, domain.StepStatusInProgress, false},
		{"SkippedToAny", domain.StepStatusSkipped, domain.StepStatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := domain.CanTransitionStep(tt.from, tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionStep(%v, %v) = %v; want %v", tt.from, tt.to, allowed, tt.allowed)
			}
			err := domain.ValidateStepTransition(tt.from, tt.to)
			if tt.allowed && err != nil {
				t.Errorf("ValidateStepTransition(%v, %v) unexpected error: %v", tt.from, tt.to, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("ValidateStepTransition(%v, %v) expected error, got nil", tt.from, tt.to)
			}
		})
	}
}
