package uploadsvc

import (
	"errors"
	"testing"
)

func TestParseUploadActionAcceptsKnownActions(t *testing.T) {
	t.Parallel()

	gotOpen, err := ParseUploadAction(string(ActionUploadOpenEndpoint))
	if err != nil {
		t.Fatalf("ParseUploadAction(open-endpoint) returned error: %v", err)
	}
	if gotOpen != ActionUploadOpenEndpoint {
		t.Fatalf("unexpected parsed open-endpoint action: %q", gotOpen)
	}

	gotReset, err := ParseUploadAction(string(ActionUploadReset))
	if err != nil {
		t.Fatalf("ParseUploadAction(reset) returned error: %v", err)
	}
	if gotReset != ActionUploadReset {
		t.Fatalf("unexpected parsed reset action: %q", gotReset)
	}
}

func TestParseUploadActionRejectsUnknownAction(t *testing.T) {
	t.Parallel()

	if _, err := ParseUploadAction("not-an-upload-action"); err == nil {
		t.Fatal("expected unknown upload action to fail parsing")
	}
}

func TestUploadPayloadPolicyContractTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		command Command
		wantErr error
	}{
		{
			name: "reset rejects open endpoint payload fields",
			command: Command{
				Action:       ActionUploadReset,
				ResetRequest: &ResetRequest{},
				OpenEndpointRequest: &OpenEndpointRequest{
					Endpoint: "ship",
				},
			},
			wantErr: ErrResetPayloadMix,
		},
		{
			name: "open-endpoint requires endpoint and token payload",
			command: Command{
				Action:              ActionUploadOpenEndpoint,
				OpenEndpointRequest: &OpenEndpointRequest{},
			},
			wantErr: ErrOpenEndpointEndpointMissing,
		},
		{
			name: "valid open-endpoint payload passes",
			command: Command{
				Action: ActionUploadOpenEndpoint,
				OpenEndpointRequest: &OpenEndpointRequest{
					Endpoint:   "ship",
					TokenID:    "token-id",
					TokenValue: "token-value",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid reset payload passes",
			command: Command{
				Action:       ActionUploadReset,
				ResetRequest: &ResetRequest{},
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCommand(tc.command)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected valid command, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil", tc.wantErr)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}
