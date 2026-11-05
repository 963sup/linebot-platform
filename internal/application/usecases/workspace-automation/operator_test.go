package workspaceautomation

import (
	"context"
	"errors"
	"testing"

	"github.com/963sup/linebot-platform/internal/domain/workspace-automation/aggregates"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/workspace-automation/value-objects"
)

type operatorAutomationRepository struct {
	findCalls int
	saveCalls int
}

func (r *operatorAutomationRepository) Find(context.Context, valueobjects.AutomationRequestID) (aggregates.AutomationRequest, error) {
	r.findCalls++
	return aggregates.AutomationRequest{}, errors.New("must not read")
}

func (r *operatorAutomationRepository) Save(context.Context, aggregates.AutomationRequest) error {
	r.saveCalls++
	return errors.New("must not write")
}

func TestOperatorExecutePreflightsProviderBeforeRepositoryAccess(t *testing.T) {
	repository := &operatorAutomationRepository{}
	operator, err := NewOperator(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operator.Execute(context.Background(), "request-1"); !errors.Is(err, ErrOperatorProviderUnavailable) {
		t.Fatalf("Execute() error = %v, want provider unavailable", err)
	}
	if repository.findCalls != 0 || repository.saveCalls != 0 {
		t.Fatalf("repository accessed before provider preflight: find=%d save=%d", repository.findCalls, repository.saveCalls)
	}
}
