package legalflow

import (
	"context"
	"fmt"
	"time"

	"github.com/example/legal-agent-loop-trace-go/internal/infrai"
)

type MatterIntakeInput struct {
	MatterID          string `json:"matter_id"`
	ClientName        string `json:"client_name"`
	OpposingParty     string `json:"opposing_party"`
	IssueSummary      string `json:"issue_summary"`
	SignedDocumentURL string `json:"signed_document_url"`
	DeadlineDate      string `json:"deadline_date"`
}

type MatterResult struct {
	MatterID         string `json:"matter_id"`
	Status           string `json:"status"`
	DeliveryStatus   string `json:"delivery_status"`
	FollowUpAction   string `json:"follow_up_action"`
	IntakeSummary    string `json:"intake_summary"`
	PromptTokenCount int    `json:"prompt_token_count"`
}

type AIClient interface {
	SummarizeMatter(ctx context.Context, input string) (string, error)
	TokensCount(ctx context.Context, input string) (int, error)
	MetricsReport(ctx context.Context, metric infrai.MetricReport) error
	ErrorsCapture(ctx context.Context, payload infrai.ErrorCapture) error
}

type MatterIntakeService struct {
	ai  AIClient
	now func() time.Time
}

func NewMatterIntakeService(ai AIClient, now func() time.Time) *MatterIntakeService {
	return &MatterIntakeService{ai: ai, now: now}
}

func (s *MatterIntakeService) ProcessMatter(ctx context.Context, input MatterIntakeInput) (MatterResult, error) {
	prompt := fmt.Sprintf(
		"Matter %s for %s against %s. %s Signed document: %s Deadline: %s.",
		input.MatterID,
		input.ClientName,
		input.OpposingParty,
		input.IssueSummary,
		input.SignedDocumentURL,
		input.DeadlineDate,
	)

	summary, err := s.ai.SummarizeMatter(ctx, prompt)
	if err != nil {
		_ = s.ai.ErrorsCapture(ctx, infrai.ErrorCapture{
			"context": map[string]any{
				"matter_id": input.MatterID,
				"stage":     "matter_intake_summary",
			},
			"exception": map[string]any{
				"message": err.Error(),
			},
		})
		return MatterResult{}, err
	}

	tokens, err := s.ai.TokensCount(ctx, prompt)
	if err != nil {
		return MatterResult{}, err
	}

	result, err := decideMatterState(input, summary, tokens, s.now())
	if err != nil {
		return MatterResult{}, err
	}

	_ = s.ai.MetricsReport(ctx, infrai.MetricReport{
		"name":  "matter_intake_prompt_tokens",
		"value": result.PromptTokenCount,
		"type":  "counter",
		"tags": map[string]string{
			"matter_id":        result.MatterID,
			"status":           result.Status,
			"delivery_status":  result.DeliveryStatus,
			"follow_up_action": result.FollowUpAction,
		},
	})

	return result, nil
}

func decideMatterState(input MatterIntakeInput, summary string, tokens int, now time.Time) (MatterResult, error) {
	deadline, err := time.Parse("2006-01-02", input.DeadlineDate)
	if err != nil {
		return MatterResult{}, fmt.Errorf("parse deadline: %w", err)
	}

	result := MatterResult{
		MatterID:         input.MatterID,
		IntakeSummary:    summary,
		PromptTokenCount: tokens,
	}

	if input.SignedDocumentURL != "" {
		result.DeliveryStatus = "signed_document_ready"
	} else {
		result.DeliveryStatus = "awaiting_signed_document"
	}

	if input.SignedDocumentURL != "" && deadline.After(now) {
		result.Status = "follow_up_scheduled"
		result.FollowUpAction = "schedule_deadline_follow_up"
		return result, nil
	}

	result.Status = "intake_in_review"
	result.FollowUpAction = "await_case_worker_review"
	return result, nil
}
