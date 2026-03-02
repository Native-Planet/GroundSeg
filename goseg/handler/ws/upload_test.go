package ws

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"groundseg/protocol/actions"
	"groundseg/structs"
	"groundseg/uploadsvc"
)

type stubUploadService struct {
	openCalls  int
	resetCalls int
	openErr    error
	resetErr   error
	lastReq    uploadsvc.OpenEndpointRequest
}

func (service *stubUploadService) OpenEndpoint(req uploadsvc.OpenEndpointRequest) error {
	service.openCalls++
	service.lastReq = req
	return service.openErr
}

func (service *stubUploadService) Reset() error {
	service.resetCalls++
	return service.resetErr
}

func buildUploadMessage(action, endpoint string, remote bool, fix bool, drive string, token structs.WsTokenStruct) []byte {
	payload := structs.WsUploadPayload{
		Token: token,
		Payload: structs.WsUploadAction{
			Action:        action,
			Endpoint:      endpoint,
			Remote:        remote,
			Fix:           fix,
			SelectedDrive: drive,
		},
	}
	data, _ := json.Marshal(payload)
	return data
}

type uploadBranchMatrixCase struct {
	name       string
	action     string
	msg        []byte
	service    *stubUploadService
	wantErr    string
	wantOpen   int
	wantReset  int
	verifyOpen bool
}

func uploadBranchMatrixCases(token structs.WsTokenStruct) []uploadBranchMatrixCase {
	return []uploadBranchMatrixCase{
		{
			name:      "decode-failure",
			action:    "",
			msg:       []byte("{invalid"),
			service:   &stubUploadService{},
			wantErr:   "Couldn't unmarshal upload payload",
			wantOpen:  0,
			wantReset: 0,
		},
		{
			name:       "open-endpoint-success",
			action:     string(actions.ActionUploadOpenEndpoint),
			msg:        buildUploadMessage(string(actions.ActionUploadOpenEndpoint), "session-matrix", true, true, "/dev/sda", token),
			service:    &stubUploadService{},
			wantOpen:   1,
			wantReset:  0,
			verifyOpen: true,
		},
		{
			name:      "open-endpoint-service-failure",
			action:    string(actions.ActionUploadOpenEndpoint),
			msg:       buildUploadMessage(string(actions.ActionUploadOpenEndpoint), "session-matrix", false, false, "", token),
			service:   &stubUploadService{openErr: errors.New("open failed")},
			wantErr:   "open upload endpoint session-matrix",
			wantOpen:  1,
			wantReset: 0,
		},
		{
			name:      "reset-success",
			action:    string(actions.ActionUploadReset),
			msg:       buildUploadMessage(string(actions.ActionUploadReset), "", false, false, "", structs.WsTokenStruct{}),
			service:   &stubUploadService{},
			wantOpen:  0,
			wantReset: 1,
		},
		{
			name:      "reset-with-extra-fields",
			action:    string(actions.ActionUploadReset),
			msg:       buildUploadMessage(string(actions.ActionUploadReset), "session-reset", true, true, "/dev/sda", structs.WsTokenStruct{}),
			service:   &stubUploadService{},
			wantErr:   "Unsupported upload action: reset command must not include open-endpoint payload",
			wantOpen:  0,
			wantReset: 0,
		},
		{
			name:      "reset-service-failure",
			action:    string(actions.ActionUploadReset),
			msg:       buildUploadMessage(string(actions.ActionUploadReset), "", false, false, "", structs.WsTokenStruct{}),
			service:   &stubUploadService{resetErr: errors.New("reset failed")},
			wantErr:   "reset upload session",
			wantOpen:  0,
			wantReset: 1,
		},
		{
			name:      "unsupported-action",
			action:    "unsupported",
			msg:       buildUploadMessage("unsupported", "", false, false, "", structs.WsTokenStruct{}),
			service:   &stubUploadService{},
			wantErr:   "Unrecognized upload action",
			wantOpen:  0,
			wantReset: 0,
		},
	}
}

func TestUploadHandlerDispatchesActions(t *testing.T) {
	service := &stubUploadService{}
	handler, err := NewUploadMessageHandler(service)
	if err != nil {
		t.Fatalf("NewUploadMessageHandler returned error: %v", err)
	}

	token := structs.WsTokenStruct{ID: "u1", Token: "tok"}
	if err := handler.Handle(buildUploadMessage(string(actions.ActionUploadOpenEndpoint), "session-1", true, true, "/dev/sda", token)); err != nil {
		t.Fatalf("open-endpoint returned error: %v", err)
	}
	if service.openCalls != 1 || service.resetCalls != 0 {
		t.Fatalf("unexpected service calls for open-endpoint: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
	if service.lastReq.Endpoint != "session-1" || !service.lastReq.Remote || !service.lastReq.Fix || service.lastReq.SelectedDrive != "/dev/sda" || service.lastReq.TokenID != token.ID || service.lastReq.TokenValue != token.Token {
		t.Fatalf("unexpected open-endpoint request: %+v", service.lastReq)
	}

	if err := handler.Handle(buildUploadMessage(string(actions.ActionUploadReset), "", false, false, "", structs.WsTokenStruct{})); err != nil {
		t.Fatalf("reset returned error: %v", err)
	}
	if service.openCalls != 1 || service.resetCalls != 1 {
		t.Fatalf("unexpected service calls after reset: open=%d reset=%d", service.openCalls, service.resetCalls)
	}
}

func TestUploadHandlerPropagatesServiceErrors(t *testing.T) {
	service := &stubUploadService{openErr: errors.New("open failed")}
	handler, err := NewUploadMessageHandler(service)
	if err != nil {
		t.Fatalf("NewUploadMessageHandler returned error: %v", err)
	}
	if err := handler.Handle(buildUploadMessage(string(actions.ActionUploadOpenEndpoint), "session-2", false, false, "", structs.WsTokenStruct{})); err == nil {
		t.Fatal("expected error when OpenEndpoint fails")
	}

	service = &stubUploadService{resetErr: errors.New("reset failed")}
	handler, err = NewUploadMessageHandler(service)
	if err != nil {
		t.Fatalf("NewUploadMessageHandler returned error: %v", err)
	}
	if err := handler.Handle(buildUploadMessage(string(actions.ActionUploadReset), "", false, false, "", structs.WsTokenStruct{})); err == nil {
		t.Fatal("expected error when Reset fails")
	}
}

func TestUploadHandlerRejectsUnknownAction(t *testing.T) {
	handler, err := NewUploadMessageHandler(&stubUploadService{})
	if err != nil {
		t.Fatalf("NewUploadMessageHandler returned error: %v", err)
	}
	if err := handler.Handle(buildUploadMessage("unknown-action", "", false, false, "", structs.WsTokenStruct{})); err == nil {
		t.Fatal("expected error for unknown upload action")
	}
}

func TestNewUploadMessageHandlerNilServiceAndUploadRejectsMalformedJSON(t *testing.T) {
	if _, err := NewUploadMessageHandler(nil); err == nil || !strings.Contains(err.Error(), "upload service is required") {
		t.Fatalf("expected constructor dependency validation error, got %v", err)
	}

	handler, err := NewUploadMessageHandler(&stubUploadService{})
	if err != nil {
		t.Fatalf("NewUploadMessageHandler returned error: %v", err)
	}
	if err := handler.Handle([]byte("{invalid")); err == nil || !strings.Contains(err.Error(), "Couldn't unmarshal upload payload") {
		t.Fatalf("expected malformed JSON error, got %v", err)
	}
}

func TestUploadHandlerBranchMatrix(t *testing.T) {
	token := structs.WsTokenStruct{ID: "u-matrix", Token: "matrix-token"}

	for _, tc := range uploadBranchMatrixCases(token) {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewUploadMessageHandler(tc.service)
			if err != nil {
				t.Fatalf("NewUploadMessageHandler returned error: %v", err)
			}
			err = handler.Handle(tc.msg)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
			}
			if tc.service.openCalls != tc.wantOpen || tc.service.resetCalls != tc.wantReset {
				t.Fatalf("unexpected service calls: open=%d reset=%d", tc.service.openCalls, tc.service.resetCalls)
			}
			if tc.verifyOpen {
				if tc.service.lastReq.Endpoint != "session-matrix" || tc.service.lastReq.TokenID != token.ID || tc.service.lastReq.TokenValue != token.Token {
					t.Fatalf("unexpected open-endpoint request: %+v", tc.service.lastReq)
				}
			}
		})
	}
}

func TestUploadHandlerBranchMatrixCoversSupportedActions(t *testing.T) {
	token := structs.WsTokenStruct{ID: "u-matrix", Token: "matrix-token"}
	type coverage struct {
		success bool
		failure bool
	}
	coverageByAction := make(map[string]coverage)

	for _, tc := range uploadBranchMatrixCases(token) {
		if tc.action == "" {
			continue
		}
		entry := coverageByAction[tc.action]
		if tc.wantErr == "" {
			entry.success = true
		} else {
			entry.failure = true
		}
		coverageByAction[tc.action] = entry
	}

	for _, action := range UploadSupportedActions() {
		entry := coverageByAction[action]
		if !entry.success || !entry.failure {
			t.Fatalf("expected success+failure branch coverage for action %q, got %+v", action, entry)
		}
	}
}
