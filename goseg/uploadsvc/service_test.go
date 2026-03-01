package uploadsvc

import (
	"errors"
	"testing"
)

type stubUploadService struct {
	lastReq    OpenEndpointRequest
	openErr    error
	resetErr   error
	openCalls  int
	resetCalls int
}

func (s *stubUploadService) OpenEndpoint(req OpenEndpointRequest) error {
	s.openCalls++
	s.lastReq = req
	return s.openErr
}

func (s *stubUploadService) Reset() error {
	s.resetCalls++
	return s.resetErr
}

func TestNewExecutorRejectsNilService(t *testing.T) {
	if _, err := NewExecutor(nil); err == nil {
		t.Fatal("expected NewExecutor to reject nil service")
	}
}

func TestExecutorDispatchesOpenEndpoint(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	cmd := Command{
		Action: ActionOpenEndpoint,
		OpenEndpointRequest: OpenEndpointRequest{
			Endpoint:      "session-a",
			TokenID:       "tok-id",
			TokenValue:    "tok-value",
			Remote:        true,
			Fix:           true,
			SelectedDrive: "/dev/sda",
		},
	}
	if err := executor.Execute(cmd); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if service.openCalls != 1 || service.resetCalls != 0 {
		t.Fatalf("unexpected call counts: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
	if service.lastReq.Endpoint != "session-a" || service.lastReq.TokenID != "tok-id" || service.lastReq.TokenValue != "tok-value" || !service.lastReq.Remote || !service.lastReq.Fix || service.lastReq.SelectedDrive != "/dev/sda" {
		t.Fatalf("unexpected request dispatched: %+v", service.lastReq)
	}
}

func TestExecutorDispatchesReset(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	if err := executor.Execute(Command{Action: ActionReset}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if service.openCalls != 0 || service.resetCalls != 1 {
		t.Fatalf("unexpected call counts: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestExecutorPropagatesOpenEndpointError(t *testing.T) {
	service := &stubUploadService{openErr: errors.New("open failed")}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	cmd := Command{
		Action: ActionOpenEndpoint,
		OpenEndpointRequest: OpenEndpointRequest{
			Endpoint:   "session-a",
			TokenID:    "tok-id",
			TokenValue: "tok-value",
		},
	}
	if err := executor.Execute(cmd); err == nil || err.Error() != "open failed" {
		t.Fatalf("expected open endpoint error to propagate, got %v", err)
	}
}

func TestExecutorPropagatesResetError(t *testing.T) {
	service := &stubUploadService{resetErr: errors.New("reset failed")}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	if err := executor.Execute(Command{Action: ActionReset}); err == nil || err.Error() != "reset failed" {
		t.Fatalf("expected reset error to propagate, got %v", err)
	}
}

func TestExecutorReturnsUnsupportedActionError(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	err = executor.Execute(Command{Action: Action("other")})
	var unsupported UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %v", err)
	}
	if unsupported.Action != Action("other") {
		t.Fatalf("unexpected unsupported action: %v", unsupported.Action)
	}
}

func TestSupportedActionsIncludesOpenEndpointAndReset(t *testing.T) {
	actions := SupportedActions()
	if len(actions) != 2 || actions[0] != ActionOpenEndpoint || actions[1] != ActionReset {
		t.Fatalf("unexpected supported actions contract: %+v", actions)
	}
}

func TestParseActionMatchesSupportedActions(t *testing.T) {
	for _, action := range SupportedActions() {
		parsed, err := ParseAction(string(action))
		if err != nil {
			t.Fatalf("expected supported action %q to parse, got error: %v", action, err)
		}
		if parsed != action {
			t.Fatalf("parse mismatch for %q: got %q", action, parsed)
		}
	}
}

func TestParseActionRejectsUnsupportedValue(t *testing.T) {
	_, err := ParseAction("unsupported")
	if err == nil {
		t.Fatal("expected ParseAction to reject unsupported value")
	}
	var unsupported UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
	}
}

func TestExecutorSupportedActionsMatchesContract(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	actions := executor.SupportedActions()
	if len(actions) != 2 || actions[0] != ActionOpenEndpoint || actions[1] != ActionReset {
		t.Fatalf("unexpected executor supported actions: %+v", actions)
	}
}

func TestExecutorDispatchTableParityAcrossSupportedActions(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}

	for _, action := range SupportedActions() {
		beforeOpen := service.openCalls
		beforeReset := service.resetCalls

		cmd := Command{Action: action}
		switch action {
		case ActionOpenEndpoint:
			cmd.OpenEndpointRequest = OpenEndpointRequest{
				Endpoint:   "matrix-endpoint",
				TokenID:    "matrix-token-id",
				TokenValue: "matrix-token",
			}
		case ActionReset:
		default:
			t.Fatalf("supported action %q has no dispatch expectation in parity test", action)
		}

		if err := executor.Execute(cmd); err != nil {
			t.Fatalf("expected supported action %q to dispatch, got error: %v", action, err)
		}

		switch action {
		case ActionOpenEndpoint:
			if service.openCalls != beforeOpen+1 || service.resetCalls != beforeReset {
				t.Fatalf("open-endpoint dispatch mismatch: open=%d reset=%d", service.openCalls-beforeOpen, service.resetCalls-beforeReset)
			}
		case ActionReset:
			if service.resetCalls != beforeReset+1 || service.openCalls != beforeOpen {
				t.Fatalf("reset dispatch mismatch: open=%d reset=%d", service.openCalls-beforeOpen, service.resetCalls-beforeReset)
			}
		}
	}
}

func TestDescribeActionCoversSupportedActions(t *testing.T) {
	service := &stubUploadService{}
	_, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	for _, action := range SupportedActions() {
		cmd := Command{Action: action}
		if action == ActionOpenEndpoint {
			cmd.OpenEndpointRequest = OpenEndpointRequest{Endpoint: "matrix-endpoint"}
		}
		operation, ok := DescribeAction(cmd)
		if !ok {
			t.Fatalf("expected DescribeAction to cover supported action %q", action)
		}
		if operation == "" {
			t.Fatalf("expected non-empty operation for action %q", action)
		}
		if action == ActionReset && operation != "reset upload session" {
			t.Fatalf("unexpected reset operation: %q", operation)
		}
		if action == ActionOpenEndpoint && operation != "open upload endpoint matrix-endpoint" {
			t.Fatalf("unexpected open-endpoint operation: %q", operation)
		}
	}
}

func TestDescribeActionReturnsUnsupportedFallback(t *testing.T) {
	operation, ok := DescribeAction(Command{Action: Action("mystery")})
	if ok {
		t.Fatalf("expected unsupported action to return no mapping, got %q", operation)
	}
	if operation != "upload action mystery" {
		t.Fatalf("expected deterministic fallback operation, got %q", operation)
	}
}
