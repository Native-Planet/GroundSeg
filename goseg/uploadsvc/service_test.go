package uploadsvc

import (
	"errors"
	"testing"

	"groundseg/protocol/actions"
)

func actionSet(values []actions.Action) map[actions.Action]struct{} {
	set := make(map[actions.Action]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func uploadContracts() []actions.UploadActionContract {
	return actions.UploadActionContracts()
}

func expectedSupportedActions() []actions.Action {
	contracts := uploadContracts()
	expected := make([]actions.Action, 0, len(contracts))
	for _, contract := range contracts {
		expected = append(expected, contract.Action)
	}
	return expected
}

func commandForContract(contract actions.UploadActionContract) Command {
	cmd := Command{Action: contract.Action}
	if contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) {
		cmd.OpenEndpointRequest = &OpenEndpointRequest{
			Endpoint:   "matrix-endpoint",
			TokenID:    "matrix-token-id",
			TokenValue: "matrix-token",
		}
	}
	if contract.RequiredPayloads.Has(actions.UploadPayloadReset) {
		cmd.ResetRequest = &ResetRequest{}
	}
	return cmd
}

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

func TestCommandFromUploadInputsSkipsOpenEndpointForResetWithoutEndpointFields(t *testing.T) {
	cmd, err := CommandFromUploadInputs(actions.ActionUploadReset, OpenEndpointRequest{}, &ResetRequest{})
	if err != nil {
		t.Fatalf("CommandFromUploadInputs returned error: %v", err)
	}
	if cmd.OpenEndpointRequest != nil {
		t.Fatalf("expected no open-endpoint request for reset without endpoint fields, got %#v", cmd.OpenEndpointRequest)
	}
}

func TestCommandFromUploadInputsRejectsResetWithOpenEndpointFields(t *testing.T) {
	_, err := CommandFromUploadInputs(actions.ActionUploadReset, OpenEndpointRequest{Endpoint: "session"}, &ResetRequest{})
	if err == nil {
		t.Fatal("expected reset payload extras to be rejected")
	}
	var validation CommandValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected CommandValidationError, got %T: %v", err, err)
	}
	if validation.Action != actions.ActionUploadReset {
		t.Fatalf("expected reset action in validation error, got %q", validation.Action)
	}
}

func TestExecutorDispatchesOpenEndpoint(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	cmd := Command{
		Action: actions.ActionUploadOpenEndpoint,
		OpenEndpointRequest: &OpenEndpointRequest{
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

func TestExecutorRejectsOpenEndpointWithoutPayload(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	if err := executor.Execute(Command{Action: actions.ActionUploadOpenEndpoint}); err == nil {
		t.Fatal("expected missing open-endpoint payload to return error")
	}
	if service.openCalls != 0 || service.resetCalls != 0 {
		t.Fatalf("unexpected service call counts: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestExecutorRejectsOpenEndpointWithResetPayload(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	err = executor.Execute(Command{
		Action: actions.ActionUploadOpenEndpoint,
		OpenEndpointRequest: &OpenEndpointRequest{
			Endpoint:   "session-a",
			TokenID:    "tok-id",
			TokenValue: "tok-value",
		},
		ResetRequest: &ResetRequest{},
	})
	if err == nil || err.Error() != "open-endpoint command must not include reset payload" {
		t.Fatalf("expected reset-payload guard, got %v", err)
	}
	if service.openCalls != 0 || service.resetCalls != 0 {
		t.Fatalf("unexpected service calls: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestExecutorDispatchesReset(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	if err := executor.Execute(Command{Action: actions.ActionUploadReset, ResetRequest: &ResetRequest{}}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if service.openCalls != 0 || service.resetCalls != 1 {
		t.Fatalf("unexpected call counts: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestExecutorRejectsResetWithOpenEndpointPayload(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	err = executor.Execute(Command{
		Action:       actions.ActionUploadReset,
		ResetRequest: &ResetRequest{},
		OpenEndpointRequest: &OpenEndpointRequest{
			Endpoint:   "session-a",
			TokenID:    "tok-id",
			TokenValue: "tok-value",
		},
	})
	if err == nil || err.Error() != "reset command must not include open-endpoint payload" {
		t.Fatalf("expected open-endpoint payload guard, got %v", err)
	}
	if service.openCalls != 0 || service.resetCalls != 0 {
		t.Fatalf("unexpected service calls: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestExecutorPropagatesOpenEndpointError(t *testing.T) {
	service := &stubUploadService{openErr: errors.New("open failed")}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	cmd := Command{
		Action: actions.ActionUploadOpenEndpoint,
		OpenEndpointRequest: &OpenEndpointRequest{
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
	if err := executor.Execute(Command{Action: actions.ActionUploadReset, ResetRequest: &ResetRequest{}}); err == nil || err.Error() != "reset failed" {
		t.Fatalf("expected reset error to propagate, got %v", err)
	}
}

func TestExecutorRejectsResetWithoutPayload(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	if err := executor.Execute(Command{Action: actions.ActionUploadReset}); err == nil {
		t.Fatal("expected missing reset payload to return error")
	}
	if service.openCalls != 0 || service.resetCalls != 0 {
		t.Fatalf("unexpected service call counts: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestExecutorReturnsUnsupportedActionError(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	err = executor.Execute(Command{Action: actions.Action("other")})
	var unsupported actions.UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %v", err)
	}
	if unsupported.Action != actions.Action("other") {
		t.Fatalf("unexpected unsupported action: %v", unsupported.Action)
	}
}

func TestSupportedActionsIncludesOpenEndpointAndReset(t *testing.T) {
	supportedActions := actions.SupportedUploadActions()
	expectedActions := expectedSupportedActions()
	if len(supportedActions) != len(expectedActions) {
		t.Fatalf("unexpected supported actions contract: %+v", supportedActions)
	}
	expected := actionSet(expectedActions)
	got := actionSet(supportedActions)
	for action := range expected {
		if _, exists := got[action]; !exists {
			t.Fatalf("expected supported actions to include %q, got %+v", action, supportedActions)
		}
	}
}

func TestParseActionMatchesSupportedActions(t *testing.T) {
	for _, action := range actions.SupportedUploadActions() {
		parsed, err := actions.ParseUploadAction(string(action))
		if err != nil {
			t.Fatalf("expected supported action %q to parse, got error: %v", action, err)
		}
		if parsed != action {
			t.Fatalf("parse mismatch for %q: got %q", action, parsed)
		}
	}
}

func TestParseActionRejectsUnsupportedValue(t *testing.T) {
	_, err := actions.ParseUploadAction("unsupported")
	if err == nil {
		t.Fatal("expected ParseAction to reject unsupported value")
	}
	var unsupported actions.UnsupportedActionError
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
	supportedActions := executor.SupportedActions()
	expectedActions := expectedSupportedActions()
	if len(supportedActions) != len(expectedActions) {
		t.Fatalf("unexpected executor supported actions: %+v", supportedActions)
	}
	expected := actionSet(expectedActions)
	got := actionSet(supportedActions)
	for action := range expected {
		if _, exists := got[action]; !exists {
			t.Fatalf("executor supported actions missing %q, got %+v", action, supportedActions)
		}
	}
}

func TestExecutorDispatchTableParityAcrossSupportedActions(t *testing.T) {
	service := &stubUploadService{}
	executor, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}

	for _, action := range actions.SupportedUploadActions() {
		beforeOpen := service.openCalls
		beforeReset := service.resetCalls

		var contract actions.UploadActionContract
		found := false
		for _, c := range uploadContracts() {
			if c.Action == action {
				contract = c
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("supported action %q is missing contract metadata", action)
		}
		cmd := commandForContract(contract)
		if cmd.Action != action {
			t.Fatalf("command contract mismatch for %q", action)
		}

		if err := executor.Execute(cmd); err != nil {
			t.Fatalf("expected supported action %q to dispatch, got error: %v", action, err)
		}

		switch {
		case contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint):
			if service.openCalls != beforeOpen+1 || service.resetCalls != beforeReset {
				t.Fatalf("open-endpoint dispatch mismatch: open=%d reset=%d", service.openCalls-beforeOpen, service.resetCalls-beforeReset)
			}
		case contract.RequiredPayloads.Has(actions.UploadPayloadReset):
			if service.resetCalls != beforeReset+1 || service.openCalls != beforeOpen {
				t.Fatalf("reset dispatch mismatch: open=%d reset=%d", service.openCalls-beforeOpen, service.resetCalls-beforeReset)
			}
		default:
			t.Fatalf("supported action %q has no dispatch expectation in parity test", action)
		}
	}
}

func TestDescribeActionCoversSupportedActions(t *testing.T) {
	service := &stubUploadService{}
	_, err := NewExecutor(service)
	if err != nil {
		t.Fatalf("NewExecutor returned error: %v", err)
	}
	for _, action := range actions.SupportedUploadActions() {
		var contract actions.UploadActionContract
		found := false
		for _, c := range uploadContracts() {
			if c.Action == action {
				contract = c
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("supported action %q is missing contract metadata", action)
		}

		cmd := commandForContract(contract)
		operation, ok := DescribeAction(cmd)
		if !ok {
			t.Fatalf("expected DescribeAction to cover supported action %q", action)
		}
		if operation == "" {
			t.Fatalf("expected non-empty operation for action %q", action)
		}

		switch {
		case contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) && operation != "open upload endpoint matrix-endpoint":
			t.Fatalf("unexpected open-endpoint operation: %q", operation)
		case contract.RequiredPayloads.Has(actions.UploadPayloadReset) && operation != "reset upload session":
			t.Fatalf("unexpected reset operation: %q", operation)
		case contract.RequiredPayloads.IsEmpty():
			if operation != string(contract.Description) {
				t.Fatalf("unexpected operation for %q: %q", action, operation)
			}
		}
	}
}

func TestDescribeActionReturnsUnsupportedFallback(t *testing.T) {
	operation, ok := DescribeAction(Command{Action: actions.Action("mystery")})
	if ok {
		t.Fatalf("expected unsupported action to return no mapping, got %q", operation)
	}
	if operation != "upload action mystery" {
		t.Fatalf("expected deterministic fallback operation, got %q", operation)
	}
}

func TestDescribeActionIsDeterministicForSupportedActions(t *testing.T) {
	for _, contract := range uploadContracts() {
		cmd := commandForContract(contract)
		first, ok := DescribeAction(cmd)
		if !ok {
			t.Fatalf("expected DescribeAction to cover %q", contract.Action)
		}
		second, ok := DescribeAction(cmd)
		if !ok {
			t.Fatalf("expected DescribeAction to cover %q", contract.Action)
		}
		if first != second {
			t.Fatalf("expected deterministic DescribeAction for %q, first=%q second=%q", contract.Action, first, second)
		}
	}
}
