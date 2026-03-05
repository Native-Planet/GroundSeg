package conformance

import (
	"errors"
	"strings"
	"testing"

	"groundseg/protocol/contracts"
	"groundseg/protocol/contracts/familycatalog"
	"groundseg/uploadsvc"
)

var actionOwnerGateTargets = map[familycatalog.OwnerModule][]string{
	familycatalog.OwnerUploadService: {"./uploadsvc/..."},
	familycatalog.OwnerSystemWiFi:    {"./system/wifi/..."},
}

func uploadCommandForBinding(spec contracts.UploadActionBindingSpec) uploadsvc.Command {
	cmd := uploadsvc.Command{Action: uploadsvc.Action(spec.Action)}
	if spec.RequiredPayloads.Has(contracts.UploadPayloadOpenEndpoint) {
		cmd.OpenEndpointRequest = &uploadsvc.OpenEndpointRequest{
			Endpoint:   "matrix-endpoint",
			TokenID:    "matrix-token-id",
			TokenValue: "matrix-token",
		}
	}
	if spec.RequiredPayloads.Has(contracts.UploadPayloadReset) {
		cmd.ResetRequest = &uploadsvc.ResetRequest{}
	}
	return cmd
}

func TestUploadRuntimeActionsMatchUploadContractRegistry(t *testing.T) {
	expectedByAction := make(map[uploadsvc.Action]contracts.UploadActionBindingSpec)
	for _, spec := range contracts.UploadActionBindingSpecs() {
		expectedByAction[uploadsvc.Action(spec.Action)] = spec
	}
	supported, err := uploadsvc.SupportedUploadActions()
	if err != nil {
		t.Fatalf("supported upload actions: %v", err)
	}
	if len(supported) != len(expectedByAction) {
		t.Fatalf("supported action count mismatch: got %d want %d", len(supported), len(expectedByAction))
	}
	contractsByAction, err := uploadsvc.UploadActionContractByAction()
	if err != nil {
		t.Fatalf("upload action contracts: %v", err)
	}
	for _, action := range supported {
		parsed, err := uploadsvc.ParseUploadAction(string(action))
		if err != nil {
			t.Fatalf("parse supported upload action %q: %v", action, err)
		}
		uploadContract, ok := contractsByAction[parsed]
		if !ok {
			t.Fatalf("missing upload action contract for %q", parsed)
		}
		expected, ok := expectedByAction[parsed]
		if !ok {
			t.Fatalf("missing upload contract registry entry for action %q", parsed)
		}
		if uploadContract.ID != expected.Contract {
			t.Fatalf("contract id mismatch for action %q: got %s want %s", parsed, uploadContract.ID, expected.Contract)
		}
		if uploadContract.Contract.Name != expected.Name || uploadContract.Contract.Description != expected.Description {
			t.Fatalf("descriptor metadata mismatch for action %q", parsed)
		}
		if uploadContract.RequiredPayloads != expected.RequiredPayloads || uploadContract.ForbiddenPayloads != expected.ForbiddenPayloads {
			t.Fatalf("payload rule mismatch for action %q", parsed)
		}
	}
}

func TestUploadActionBindingsValidateRuntimePayloadRules(t *testing.T) {
	specs := contracts.UploadActionBindingSpecs()
	if len(specs) == 0 {
		t.Fatal("expected upload action binding specs")
	}

	for _, spec := range specs {
		spec := spec
		t.Run(string(spec.Action), func(t *testing.T) {
			valid := uploadCommandForBinding(spec)
			if err := uploadsvc.ValidateCommand(valid); err != nil {
				t.Fatalf("expected contract-valid payload to pass for action %q: %v", spec.Action, err)
			}

			if spec.RequiredPayloads.Has(contracts.UploadPayloadOpenEndpoint) {
				missingOpen := valid
				missingOpen.OpenEndpointRequest = nil
				err := uploadsvc.ValidateCommand(missingOpen)
				if !errors.Is(err, uploadsvc.ErrOpenEndpointRequestMissing) {
					t.Fatalf("expected missing open-endpoint payload guard for %q, got %v", spec.Action, err)
				}
			}

			if spec.RequiredPayloads.Has(contracts.UploadPayloadReset) {
				missingReset := valid
				missingReset.ResetRequest = nil
				err := uploadsvc.ValidateCommand(missingReset)
				if !errors.Is(err, uploadsvc.ErrResetRequestMissing) {
					t.Fatalf("expected missing reset payload guard for %q, got %v", spec.Action, err)
				}
			}

			if spec.ForbiddenPayloads.Has(contracts.UploadPayloadOpenEndpoint) {
				withForbiddenOpen := valid
				withForbiddenOpen.OpenEndpointRequest = &uploadsvc.OpenEndpointRequest{
					Endpoint:   "forbidden-endpoint",
					TokenID:    "forbidden-token-id",
					TokenValue: "forbidden-token",
				}
				err := uploadsvc.ValidateCommand(withForbiddenOpen)
				if !errors.Is(err, uploadsvc.ErrResetPayloadMix) {
					t.Fatalf("expected forbidden open-endpoint payload guard for %q, got %v", spec.Action, err)
				}
			}

			if spec.ForbiddenPayloads.Has(contracts.UploadPayloadReset) {
				withForbiddenReset := valid
				withForbiddenReset.ResetRequest = &uploadsvc.ResetRequest{}
				err := uploadsvc.ValidateCommand(withForbiddenReset)
				if !errors.Is(err, uploadsvc.ErrOpenEndpointPayloadMix) {
					t.Fatalf("expected forbidden reset payload guard for %q, got %v", spec.Action, err)
				}
			}
		})
	}
}

func TestUploadOpenEndpointPayloadRequiresCompleteIdentityFields(t *testing.T) {
	specs := contracts.UploadActionBindingSpecs()
	for _, spec := range specs {
		if !spec.RequiredPayloads.Has(contracts.UploadPayloadOpenEndpoint) {
			continue
		}

		base := uploadCommandForBinding(spec)
		cases := []struct {
			name   string
			mutate func(*uploadsvc.Command)
			want   error
		}{
			{
				name: "missing endpoint",
				mutate: func(cmd *uploadsvc.Command) {
					cmd.OpenEndpointRequest.Endpoint = ""
				},
				want: uploadsvc.ErrOpenEndpointEndpointMissing,
			},
			{
				name: "missing token id",
				mutate: func(cmd *uploadsvc.Command) {
					cmd.OpenEndpointRequest.TokenID = ""
				},
				want: uploadsvc.ErrOpenEndpointTokenIDMissing,
			},
			{
				name: "missing token value",
				mutate: func(cmd *uploadsvc.Command) {
					cmd.OpenEndpointRequest.TokenValue = ""
				},
				want: uploadsvc.ErrOpenEndpointTokenValueMissing,
			},
		}

		for _, tc := range cases {
			t.Run(string(spec.Action)+"_"+tc.name, func(t *testing.T) {
				cmd := base
				if cmd.OpenEndpointRequest == nil {
					t.Fatalf("expected open-endpoint request for action %q", spec.Action)
				}
				mutated := *cmd.OpenEndpointRequest
				cmd.OpenEndpointRequest = &mutated
				tc.mutate(&cmd)
				err := uploadsvc.ValidateCommand(cmd)
				if !errors.Is(err, tc.want) {
					t.Fatalf("expected %v for %q, got %v", tc.want, spec.Action, err)
				}
			})
		}
	}
}

func TestActionContractOwnersHaveRuntimeGateTargets(t *testing.T) {
	seenOwners := map[familycatalog.OwnerModule]struct{}{}
	for _, spec := range familycatalog.AllActionSpecs() {
		owner := familycatalog.OwnerModule(spec.Owner)
		if strings.TrimSpace(string(owner)) == "" {
			t.Fatalf("action %s:%s has empty owner", spec.Namespace, spec.Action)
		}
		targets, ok := actionOwnerGateTargets[owner]
		if !ok {
			t.Fatalf("owner %q has no quality-gate test target mapping", owner)
		}
		if len(targets) == 0 {
			t.Fatalf("owner %q has an empty quality-gate test target list", owner)
		}
		for _, target := range targets {
			if strings.TrimSpace(target) == "" {
				t.Fatalf("owner %q has blank quality-gate test target entry", owner)
			}
		}
		seenOwners[owner] = struct{}{}
	}

	for owner := range actionOwnerGateTargets {
		if _, ok := seenOwners[owner]; !ok {
			t.Fatalf("quality-gate owner mapping contains unused owner %q", owner)
		}
	}
}
