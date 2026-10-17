package legalflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/legal-agent-loop-trace-go/internal/infrai"
)

type stubAI struct {
	summary      string
	summaryErr   error
	tokens       int
	captureCalls int
}

func (s *stubAI) SummarizeMatter(ctx context.Context, input string) (string, error) {
	return s.summary, s.summaryErr
}

func (s *stubAI) TokensCount(ctx context.Context, input string) (int, error) {
	return s.tokens, nil
}

func (s *stubAI) MetricsReport(ctx context.Context, metric infrai.MetricReport) error {
	return nil
}

func (s *stubAI) ErrorsCapture(ctx context.Context, payload infrai.ErrorCapture) error {
	s.captureCalls++
	return nil
}

func TestProcessMatterDecisionTable(t *testing.T) {
	now := func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name               string
		input              MatterIntakeInput
		expectedStatus     string
		expectedDelivery   string
		expectedFollowUp   string
		expectedTokenCount int
	}{
		{
			name: "signed document and future deadline schedules follow up",
			input: MatterIntakeInput{
				MatterID:          "MAT-1001",
				ClientName:        "Dana Ortiz",
				OpposingParty:     "North Pier Builders",
				IssueSummary:      "Signed settlement arrived.",
				SignedDocumentURL: "https://example.com/docs/MAT-1001-settlement.pdf",
				DeadlineDate:      "2026-04-12",
			},
			expectedStatus:     "follow_up_scheduled",
			expectedDelivery:   "signed_document_ready",
			expectedFollowUp:   "schedule_deadline_follow_up",
			expectedTokenCount: 42,
		},
		{
			name: "missing signed document stays in review",
			input: MatterIntakeInput{
				MatterID:      "MAT-1002",
				ClientName:    "Dana Ortiz",
				OpposingParty: "North Pier Builders",
				IssueSummary:  "Need confirmation from client.",
				DeadlineDate:  "2026-04-12",
			},
			expectedStatus:     "intake_in_review",
			expectedDelivery:   "awaiting_signed_document",
			expectedFollowUp:   "await_case_worker_review",
			expectedTokenCount: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMatterIntakeService(&stubAI{summary: "Settlement received; confirm delivery and deadline follow-up.", tokens: 42}, now)
			result, err := service.ProcessMatter(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("ProcessMatter() error = %v", err)
			}
			if result.Status != tt.expectedStatus {
				t.Fatalf("status = %q, want %q", result.Status, tt.expectedStatus)
			}
			if result.DeliveryStatus != tt.expectedDelivery {
				t.Fatalf("delivery_status = %q, want %q", result.DeliveryStatus, tt.expectedDelivery)
			}
			if result.FollowUpAction != tt.expectedFollowUp {
				t.Fatalf("follow_up_action = %q, want %q", result.FollowUpAction, tt.expectedFollowUp)
			}
			if result.PromptTokenCount != tt.expectedTokenCount {
				t.Fatalf("prompt_token_count = %d, want %d", result.PromptTokenCount, tt.expectedTokenCount)
			}
		})
	}
}

func TestProcessMatterCapturesSummaryFailure(t *testing.T) {
	ai := &stubAI{summaryErr: errors.New("summary failed")}
	service := NewMatterIntakeService(ai, func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	})

	_, err := service.ProcessMatter(context.Background(), MatterIntakeInput{
		MatterID:     "MAT-1003",
		ClientName:   "Dana Ortiz",
		OpposingParty:"North Pier Builders",
		IssueSummary: "Signed settlement arrived.",
		DeadlineDate: "2026-04-12",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if ai.captureCalls != 1 {
		t.Fatalf("captureCalls = %d, want 1", ai.captureCalls)
	}
}
