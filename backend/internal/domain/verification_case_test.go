package domain

import (
	"errors"
	"testing"
)

func TestVerificationCase_TransitionTo(t *testing.T) {
	statuses := []CaseStatus{StatusDraft, StatusSubmitted, StatusInReview, StatusApproved, StatusRejected}

	legal := map[CaseStatus][]CaseStatus{
		StatusDraft:     {StatusSubmitted},
		StatusSubmitted: {StatusInReview},
		StatusInReview:  {StatusApproved, StatusRejected},
	}
	isLegal := func(from, to CaseStatus) bool {
		for _, s := range legal[from] {
			if s == to {
				return true
			}
		}
		return false
	}

	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+" to "+string(to), func(t *testing.T) {
				c := &VerificationCase{Status: from}

				err := c.TransitionTo(to)

				if isLegal(from, to) {
					if err != nil {
						t.Fatalf("TransitionTo() error = %v, want nil", err)
					}
					if c.Status != to {
						t.Fatalf("status = %s, want %s", c.Status, to)
					}
					return
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Fatalf("TransitionTo() error = %v, want ErrInvalidTransition", err)
				}
				if c.Status != from {
					t.Fatalf("status changed to %s on an illegal transition, want %s", c.Status, from)
				}
			})
		}
	}
}

func TestCaseStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status CaseStatus
		want   bool
	}{
		{StatusDraft, false},
		{StatusSubmitted, false},
		{StatusInReview, false},
		{StatusApproved, true},
		{StatusRejected, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Fatalf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}
