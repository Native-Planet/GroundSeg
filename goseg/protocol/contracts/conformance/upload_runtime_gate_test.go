package conformance

import (
	"errors"
	"fmt"
	"testing"

	"groundseg/protocol/contracts/governance"
	"groundseg/uploadsvc"
)

func TestUploadRuntimeMatchesGovernanceDeclarations(t *testing.T) {
	contractsByAction, err := uploadsvc.UploadActionContractByAction()
	if err != nil {
		t.Fatalf("load upload action contracts: %v", err)
	}

	declarations := governance.UploadActionDeclarations()
	if len(declarations) == 0 {
		t.Fatal("expected governance upload declarations")
	}

	for _, declaration := range declarations {
		action := uploadsvc.Action(declaration.Action)
		t.Run(string(action), func(t *testing.T) {
			parsed, err := uploadsvc.ParseUploadAction(declaration.Action)
			if err != nil {
				t.Fatalf("parse upload action %q: %v", declaration.Action, err)
			}
			if parsed != action {
				t.Fatalf("parsed action mismatch: got %q want %q", parsed, action)
			}

			uploadContract, ok := contractsByAction[action]
			if !ok {
				t.Fatalf("missing upload contract for action %q", action)
			}
			if string(uploadContract.ID) != declaration.ContractID {
				t.Fatalf("contract id mismatch: got %s want %s", uploadContract.ID, declaration.ContractID)
			}

			valid := validUploadCommand(action, declaration)
			if err := uploadsvc.ValidateCommand(valid); err != nil {
				t.Fatalf("valid governance-aligned command rejected: %v", err)
			}

			assertRequiredPayloadValidation(t, action, declaration)
			assertForbiddenPayloadValidation(t, action, declaration)
		})
	}
}

func validUploadCommand(action uploadsvc.Action, declaration governance.ActionDeclaration) uploadsvc.Command {
	command := uploadsvc.Command{
		Action: action,
	}
	if declaration.RequiredPayloads.Has(governance.UploadPayloadRuleOpenEndpoint) {
		command.OpenEndpointRequest = &uploadsvc.OpenEndpointRequest{
			Endpoint:   "ship",
			TokenID:    "token-id",
			TokenValue: "token-value",
		}
	}
	if declaration.RequiredPayloads.Has(governance.UploadPayloadRuleReset) {
		command.ResetRequest = &uploadsvc.ResetRequest{}
	}
	return command
}

func assertRequiredPayloadValidation(t *testing.T, action uploadsvc.Action, declaration governance.ActionDeclaration) {
	t.Helper()

	if declaration.RequiredPayloads.Has(governance.UploadPayloadRuleOpenEndpoint) {
		command := validUploadCommand(action, declaration)
		command.OpenEndpointRequest = nil
		err := uploadsvc.ValidateCommand(command)
		if !errors.Is(err, uploadsvc.ErrOpenEndpointRequestMissing) {
			t.Fatalf("missing open-endpoint payload should fail with ErrOpenEndpointRequestMissing, got: %v", err)
		}
	}

	if declaration.RequiredPayloads.Has(governance.UploadPayloadRuleReset) {
		command := validUploadCommand(action, declaration)
		command.ResetRequest = nil
		err := uploadsvc.ValidateCommand(command)
		if !errors.Is(err, uploadsvc.ErrResetRequestMissing) {
			t.Fatalf("missing reset payload should fail with ErrResetRequestMissing, got: %v", err)
		}
	}
}

func assertForbiddenPayloadValidation(t *testing.T, action uploadsvc.Action, declaration governance.ActionDeclaration) {
	t.Helper()

	if declaration.ForbiddenPayloads.Has(governance.UploadPayloadRuleOpenEndpoint) {
		command := validUploadCommand(action, declaration)
		command.OpenEndpointRequest = &uploadsvc.OpenEndpointRequest{
			Endpoint:   "ship",
			TokenID:    "token-id",
			TokenValue: "token-value",
		}
		err := uploadsvc.ValidateCommand(command)
		if !errors.Is(err, uploadsvc.ErrResetPayloadMix) {
			t.Fatalf("forbidden open-endpoint payload should fail with ErrResetPayloadMix, got: %v", err)
		}
	}

	if declaration.ForbiddenPayloads.Has(governance.UploadPayloadRuleReset) {
		command := validUploadCommand(action, declaration)
		command.ResetRequest = &uploadsvc.ResetRequest{}
		err := uploadsvc.ValidateCommand(command)
		if !errors.Is(err, uploadsvc.ErrOpenEndpointPayloadMix) {
			t.Fatalf("forbidden reset payload should fail with ErrOpenEndpointPayloadMix, got: %v", err)
		}
	}
}

func TestUploadRuntimeHasNoUndeclaredGovernanceActions(t *testing.T) {
	contractsByAction, err := uploadsvc.UploadActionContractByAction()
	if err != nil {
		t.Fatalf("load upload action contracts: %v", err)
	}
	declared := make(map[uploadsvc.Action]struct{})
	for _, declaration := range governance.UploadActionDeclarations() {
		declared[uploadsvc.Action(declaration.Action)] = struct{}{}
	}
	for action := range contractsByAction {
		if _, ok := declared[action]; ok {
			continue
		}
		t.Fatalf("upload runtime action %q missing governance declaration", action)
	}
	if len(contractsByAction) != len(declared) {
		t.Fatalf(
			"governance/upload runtime action count drift: runtime=%d governance=%d",
			len(contractsByAction),
			len(declared),
		)
	}

	for action := range declared {
		if _, ok := contractsByAction[action]; ok {
			continue
		}
		t.Fatalf("governance action %q missing upload runtime contract", action)
	}
}

func TestGovernanceUploadDeclarationsBelongToUploadNamespace(t *testing.T) {
	for _, declaration := range governance.UploadActionDeclarations() {
		if declaration.Namespace != governance.NamespaceUpload {
			t.Fatalf("unexpected namespace in upload declaration %s: %s", declaration.Action, declaration.Namespace)
		}
		expectedPrefix := fmt.Sprintf("%s.%s.", governance.ProtocolActionContractRoot, governance.NamespaceUpload)
		if len(declaration.ContractID) < len(expectedPrefix) || declaration.ContractID[:len(expectedPrefix)] != expectedPrefix {
			t.Fatalf("upload contract id %q does not use upload namespace prefix %q", declaration.ContractID, expectedPrefix)
		}
	}
}
