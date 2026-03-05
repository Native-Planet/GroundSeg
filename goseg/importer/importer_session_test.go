package importer

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"groundseg/auth/tokens"
	"groundseg/structs"
)

func TestOpenUploadEndpointRejectsInvalidSessionKey(t *testing.T) {
	err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: "not-a-valid-key",
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		SelectedDrive: "system-drive",
	})
	if err == nil {
		t.Fatal("expected invalid session key error")
	}
	if !strings.Contains(err.Error(), "invalid upload session key format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenUploadEndpointCreatesAndUpdatesSystemDriveSession(t *testing.T) {
	resetUploadSessionsForTest(t)
	runtime := authorizeTokenIDsForTest(t, "token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	createCmd := OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		Remote:        false,
		Fix:           false,
		SelectedDrive: "system-drive",
	}
	if err := OpenUploadEndpoint(createCmd, runtime); err != nil {
		t.Fatalf("OpenUploadEndpoint(create) returned error: %v", err)
	}

	uploadMu.Lock()
	createdSession, exists := uploadSessions[endpoint]
	uploadMu.Unlock()
	if !exists {
		t.Fatalf("expected upload session %s to be created", endpoint)
	}
	if createdSession.NeedsFormatting {
		t.Fatal("system-drive session should not require formatting")
	}
	if createdSession.Remote || createdSession.Fix {
		t.Fatalf("unexpected initial flags in created session: %+v", createdSession)
	}

	updateCmd := createCmd
	updateCmd.Remote = true
	updateCmd.Fix = true
	if err := OpenUploadEndpoint(updateCmd, runtime); err != nil {
		t.Fatalf("OpenUploadEndpoint(update) returned error: %v", err)
	}

	uploadMu.Lock()
	updated := uploadSessions[endpoint]
	uploadMu.Unlock()
	if !updated.Remote || !updated.Fix {
		t.Fatalf("expected updated session flags to be true, got %+v", updated)
	}
}

func TestOpenUploadEndpointRejectsTokenMismatchOnExistingSession(t *testing.T) {
	resetUploadSessionsForTest(t)
	runtime := authorizeTokenIDsForTest(t, "token-id", "other-token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	if err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		SelectedDrive: "system-drive",
	}, runtime); err != nil {
		t.Fatalf("OpenUploadEndpoint(create) returned error: %v", err)
	}

	err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "other-token-id",
			Token: "other-token-value",
		},
		SelectedDrive: "system-drive",
	}, runtime)
	if err == nil {
		t.Fatal("expected token mismatch error")
	}
	if !strings.Contains(err.Error(), "upload token validation failed: upload token does not match upload session") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenUploadEndpointPassesRequestHeadersToValidator(t *testing.T) {
	resetUploadSessionsForTest(t)

	var observedRequest *http.Request
	runtime := defaultImporterRuntime()
	runtime.ValidateUploadSessionTokenFn = func(sessionToken structs.WsTokenStruct, providedToken structs.WsTokenStruct, request *http.Request) tokens.UploadTokenAuthorizationResult {
		observedRequest = request
		if request == nil {
			return tokens.UploadTokenAuthorizationResult{
				Status:           tokens.UploadValidationStatusTokenContract,
				AuthorizationErr: fmt.Errorf("request should not be nil"),
			}
		}
		if got := request.Header.Get("X-Upload-Token-Id"); got != providedToken.ID {
			return tokens.UploadTokenAuthorizationResult{
				Status:           tokens.UploadValidationStatusTokenContract,
				AuthorizationErr: fmt.Errorf("unexpected token id header %q", got),
			}
		}
		if got := request.Header.Get("X-Upload-Token"); got != providedToken.Token {
			return tokens.UploadTokenAuthorizationResult{
				Status:           tokens.UploadValidationStatusTokenContract,
				AuthorizationErr: fmt.Errorf("unexpected token hash header %q", got),
			}
		}
		return tokens.UploadTokenAuthorizationResult{
			Status:          tokens.UploadValidationStatusAuthorized,
			AuthorizedToken: providedToken.Token,
		}
	}

	err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: "0123456789abcdef0123456789abcdef",
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		SelectedDrive: "system-drive",
	}, runtime)
	if err != nil {
		t.Fatalf("OpenUploadEndpoint returned error: %v", err)
	}
	if observedRequest == nil {
		t.Fatal("expected validator to receive a request with token headers")
	}
}

func TestSetUploadSessionWrapsOpenUploadEndpoint(t *testing.T) {
	resetUploadSessionsForTest(t)
	runtime := authorizeTokenIDsForTest(t, "token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	payload := structs.WsUploadPayload{
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		Payload: structs.WsUploadAction{
			Endpoint:      endpoint,
			Remote:        true,
			Fix:           true,
			SelectedDrive: "system-drive",
		},
	}
	if err := SetUploadSession(payload, runtime); err != nil {
		t.Fatalf("SetUploadSession returned error: %v", err)
	}

	uploadMu.Lock()
	sesh, exists := uploadSessions[endpoint]
	uploadMu.Unlock()
	if !exists {
		t.Fatalf("expected upload session %s to be present", endpoint)
	}
	if !sesh.Remote || !sesh.Fix {
		t.Fatalf("expected remote+fix true in session, got %+v", sesh)
	}
}
