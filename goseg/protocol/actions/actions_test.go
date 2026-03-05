package actions

import (
	"encoding/json"
	"errors"
	"groundseg/protocol/contracts"
	"reflect"
	"testing"
	"time"
)

func versionBefore(reference string) string {
	referenceTime, err := time.Parse("2006.01.02", reference)
	if err != nil {
		return reference
	}
	return referenceTime.Add(-24 * time.Hour).Format("2006.01.02")
}

func mustSupportedActions(t *testing.T, namespace Namespace) []Action {
	t.Helper()
	actions, err := SupportedActions(namespace)
	if err != nil {
		t.Fatalf("supported actions for %s: %v", namespace, err)
	}
	return actions
}

func mustActionMetas(t *testing.T, namespace Namespace) []ActionMeta {
	t.Helper()
	metas, err := ActionMetas(namespace)
	if err != nil {
		t.Fatalf("action metadata for %s: %v", namespace, err)
	}
	return metas
}

func mustUploadActionContracts(t *testing.T) []UploadActionContract {
	t.Helper()
	contracts, err := UploadActionContracts()
	if err != nil {
		t.Fatalf("upload action contracts: %v", err)
	}
	return contracts
}

func mustUploadActionContract(t *testing.T, action Action) UploadActionContract {
	t.Helper()
	contract, err := UploadActionContractForAction(action)
	if err != nil {
		t.Fatalf("upload action contract for %s: %v", action, err)
	}
	return contract
}

func mustContractForAction(t *testing.T, namespace Namespace, action Action) ActionContract {
	t.Helper()
	contract, err := ContractForAction(namespace, action)
	if err != nil {
		t.Fatalf("contract for %s:%s: %v", namespace, action, err)
	}
	return contract
}

func mustContractMetadata(t *testing.T, namespace Namespace, action Action) contracts.ContractMetadata {
	t.Helper()
	metadata, err := ContractMetadataForAction(namespace, action)
	if err != nil {
		t.Fatalf("contract metadata for %s:%s: %v", namespace, action, err)
	}
	return metadata
}

func mustActionActive(t *testing.T, namespace Namespace, version string, action Action) bool {
	t.Helper()
	active, err := IsActionContractActive(namespace, version, action)
	if err != nil {
		t.Fatalf("action active for %s:%s at %s: %v", namespace, action, version, err)
	}
	return active
}

func mustActionDeprecated(t *testing.T, namespace Namespace, version string, action Action) bool {
	t.Helper()
	deprecated, err := IsActionContractDeprecated(namespace, version, action)
	if err != nil {
		t.Fatalf("action deprecated for %s:%s at %s: %v", namespace, action, version, err)
	}
	return deprecated
}

func TestParseActionValidatesNamespace(t *testing.T) {
	got, err := ParseAction(NamespaceUpload, string(ActionUploadReset))
	if err != nil {
		t.Fatalf("expected supported action to parse, got %v", err)
	}
	if got != ActionUploadReset {
		t.Fatalf("unexpected action: %q", got)
	}
}

func TestParseActionRejectsUnknownAction(t *testing.T) {
	_, err := ParseAction(NamespaceUpload, "bogus")
	if err == nil {
		t.Fatalf("expected unknown action to fail")
	}
	var unsupported UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
	}
	if unsupported.Namespace != NamespaceUpload {
		t.Fatalf("expected upload namespace, got %s", unsupported.Namespace)
	}
}

func TestParseActionRejectsActionFromOtherNamespace(t *testing.T) {
	_, err := ParseAction(NamespaceUpload, string(ActionC2CConnect))
	if err == nil {
		t.Fatalf("expected c2c action to be rejected by upload namespace")
	}
	var unsupported UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
	}
	if unsupported.Namespace != NamespaceUpload {
		t.Fatalf("expected unsupported action namespace to be upload, got %s", unsupported.Namespace)
	}
	if unsupported.Action != ActionC2CConnect {
		t.Fatalf("expected unsupported action value %q, got %q", ActionC2CConnect, unsupported.Action)
	}
}

func TestParseUploadActionReportsDeterministicErrorType(t *testing.T) {
	_, err := ParseAction(NamespaceUpload, "definitely-not-upload")
	if err == nil {
		t.Fatal("expected unknown upload action to be rejected")
	}
	var unsupported UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
	}
}

func TestSupportedActionsUnknownNamespaceReturnsNil(t *testing.T) {
	if _, err := SupportedActions("invalid"); err == nil {
		t.Fatal("expected unsupported namespace error")
	}
}

func TestActionMetasReturnsContractsByNamespace(t *testing.T) {
	upload := mustActionMetas(t, NamespaceUpload)
	expectedUpload := []ActionMeta{
		{Action: ActionUploadOpenEndpoint, Description: "open upload endpoint"},
		{Action: ActionUploadReset, Description: "reset upload session"},
	}
	if !reflect.DeepEqual(upload, expectedUpload) {
		t.Fatalf("unexpected upload action metadata: got %#v", upload)
	}

	c2c := mustActionMetas(t, NamespaceC2C)
	expectedC2C := []ActionMeta{
		{Action: ActionC2CConnect, Description: "connect c2c client"},
	}
	if !reflect.DeepEqual(c2c, expectedC2C) {
		t.Fatalf("unexpected c2c action metadata: got %#v", c2c)
	}
}

func TestActionMetasForInvalidNamespaceReturnsNil(t *testing.T) {
	if _, err := ActionMetas("invalid"); err == nil {
		t.Fatal("expected unsupported namespace error")
	}
}

func TestActionMetasReturnsCopyByValue(t *testing.T) {
	first := mustActionMetas(t, NamespaceUpload)
	if len(first) == 0 {
		t.Fatal("expected at least one upload action metadata entry")
	}
	if first[0].Action != ActionUploadOpenEndpoint {
		t.Fatalf("precondition failed: first upload action %q", first[0].Action)
	}
	first[0].Action = "mutated"
	first[0].Description = "mutated description"

	second := mustActionMetas(t, NamespaceUpload)
	if second[0].Action == first[0].Action {
		t.Fatalf("expected ActionMetas return value to be copied by value, got mutated action %q", second[0].Action)
	}
	if second[0].Description == "mutated description" {
		t.Fatalf("expected ActionMetas return value to be insulated from external mutation")
	}
}

func TestSupportedUploadActionsMatchesContract(t *testing.T) {
	got := mustSupportedActions(t, NamespaceUpload)
	expected := []Action{ActionUploadOpenEndpoint, ActionUploadReset}
	if len(got) != len(expected) {
		t.Fatalf("expected %d upload actions, got %d", len(expected), len(got))
	}
	for i, action := range expected {
		if got[i] != action {
			t.Fatalf("expected upload action %q at index %d, got %q", action, i, got[i])
		}
		if _, err := ParseAction(NamespaceUpload, string(action)); err != nil {
			t.Fatalf("expected action %q to parse as supported: %v", action, err)
		}
	}
}

func TestUploadActionContractsIncludesExpectedActions(t *testing.T) {
	got := mustUploadActionContracts(t)
	if len(got) != 2 {
		t.Fatalf("expected 2 upload action contracts, got %d", len(got))
	}

	got[0].Action = "mutated"
	gotCopy := mustUploadActionContracts(t)
	if gotCopy[0].Action == "mutated" {
		t.Fatalf("expected upload action contracts to be copied by value")
	}
}

func TestUploadActionContractsAreOrdered(t *testing.T) {
	got := mustUploadActionContracts(t)
	want := []UploadActionContract{
		{
			ActionContract: ActionContract{
				Action:      ActionUploadOpenEndpoint,
				ID:          contracts.UploadOpenEndpointAction,
				Description: "open upload endpoint",
				Contract: contracts.ContractDescriptor{
					Description: "open upload endpoint",
					Name:        "UploadOpenEndpointAction",
					ContractMetadata: contracts.ContractMetadata{
						IntroducedIn:  "2026.03.02",
						Compatibility: contracts.CompatibilityBackwardSafe,
					},
				},
			},
			RequiredPayloads:  UploadPayloadOpenEndpoint,
			ForbiddenPayloads: UploadPayloadReset,
		},
		{
			ActionContract: ActionContract{
				Action:      ActionUploadReset,
				ID:          contracts.UploadResetAction,
				Description: "reset upload session",
				Contract: contracts.ContractDescriptor{
					Description: "reset upload session",
					Name:        "UploadResetAction",
					ContractMetadata: contracts.ContractMetadata{
						IntroducedIn:  "2026.03.02",
						Compatibility: contracts.CompatibilityBackwardSafe,
					},
				},
			},
			RequiredPayloads:  UploadPayloadReset,
			ForbiddenPayloads: UploadPayloadOpenEndpoint,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected upload action contract ordering: got %#v", got)
	}
}

func TestUploadActionContractForActionResolvesKnownAction(t *testing.T) {
	for _, action := range []Action{ActionUploadOpenEndpoint, ActionUploadReset} {
		t.Run(string(action), func(t *testing.T) {
			contract := mustUploadActionContract(t, action)
			if contract.Action != action {
				t.Fatalf("unexpected contract action: %v", contract.Action)
			}
		})
	}
}

func TestUploadActionContractForActionUnknownAction(t *testing.T) {
	_, err := UploadActionContractForAction("upload-action-does-not-exist")
	if err == nil {
		t.Fatal("expected unknown upload action contract lookup to fail")
	}
}

func TestUploadActionContractForActionReturnsCopy(t *testing.T) {
	got := mustUploadActionContracts(t)
	if len(got) == 0 {
		t.Fatal("expected at least one upload action contract")
	}
	contract := got[0]
	contract.Action = "mutated"

	rechecked := mustUploadActionContract(t, got[0].Action)
	if rechecked.Action != got[0].Action {
		t.Fatalf("expected contract copy lookup to stay stable, got %q", rechecked.Action)
	}
}

func TestSupportedC2CActionsMatchesContract(t *testing.T) {
	got := mustSupportedActions(t, NamespaceC2C)
	expected := []Action{ActionC2CConnect}
	if len(got) != len(expected) {
		t.Fatalf("expected %d c2c actions, got %d", len(expected), len(got))
	}
	for i, action := range expected {
		if got[i] != action {
			t.Fatalf("expected c2c action %q at index %d, got %q", action, i, got[i])
		}
	}
}

func TestUploadActionMetadataContainsLifecycle(t *testing.T) {
	for _, action := range []Action{ActionUploadOpenEndpoint, ActionUploadReset} {
		meta := mustContractMetadata(t, NamespaceUpload, action)
		if meta.IntroducedIn == "" {
			t.Fatalf("expected introduced version for %q", action)
		}
		if meta.Compatibility == "" {
			t.Fatalf("expected compatibility for %q", action)
		}
	}
}

func TestC2CActionMetadataContainsLifecycle(t *testing.T) {
	meta := mustContractMetadata(t, NamespaceC2C, ActionC2CConnect)
	if meta.IntroducedIn == "" {
		t.Fatal("expected introduced version for C2C connect action")
	}
	if meta.Compatibility == "" {
		t.Fatal("expected compatibility for C2C connect action")
	}
}

func TestUploadActionCompatibilityChecksUseMetadata(t *testing.T) {
	if !mustActionActive(t, NamespaceUpload, CurrentContractVersion, ActionUploadOpenEndpoint) {
		t.Fatal("expected upload open-endpoint action to be active at current version")
	}
	if mustActionDeprecated(t, NamespaceUpload, CurrentContractVersion, ActionUploadOpenEndpoint) {
		t.Fatal("did not expect action to be deprecated at current version")
	}
	if mustActionActive(t, NamespaceUpload, "2026.01.01", ActionUploadOpenEndpoint) {
		t.Fatal("did not expect action to be active before introduction date")
	}
}

func TestC2CActionContractForActionResolvesKnownAction(t *testing.T) {
	contract := mustContractForAction(t, NamespaceC2C, ActionC2CConnect)
	if contract.Action != ActionC2CConnect {
		t.Fatalf("unexpected C2C contract action: %q", contract.Action)
	}
}

func TestC2CActionCompatibilityChecksUseMetadata(t *testing.T) {
	if !mustActionActive(t, NamespaceC2C, CurrentContractVersion, ActionC2CConnect) {
		t.Fatal("expected C2C connect action to be active at current version")
	}
	if mustActionDeprecated(t, NamespaceC2C, CurrentContractVersion, ActionC2CConnect) {
		t.Fatal("did not expect action to be deprecated at current version")
	}
	if mustActionActive(t, NamespaceC2C, "2026.01.01", ActionC2CConnect) {
		t.Fatal("did not expect C2C connect action to be active before introduction date")
	}
}

func TestUploadActionCompatibilityBoundaryCases(t *testing.T) {
	if mustActionActive(t, NamespaceUpload, "", ActionUploadOpenEndpoint) {
		t.Fatal("expected empty version to be inactive")
	}
	if mustActionActive(t, NamespaceUpload, "2026.01.01", ActionUploadOpenEndpoint) {
		t.Fatal("expected open-endpoint to be inactive before introduction")
	}
	if mustActionActive(t, NamespaceUpload, "bad-version", ActionUploadOpenEndpoint) {
		t.Fatal("expected malformed version to be inactive")
	}
	if mustActionDeprecated(t, NamespaceUpload, CurrentContractVersion, ActionUploadOpenEndpoint) {
		t.Fatal("expected current action not to be deprecated")
	}
	if mustActionDeprecated(t, NamespaceUpload, "", ActionUploadOpenEndpoint) {
		t.Fatal("expected empty version not to be deprecated")
	}
	if !mustActionActive(t, NamespaceUpload, CurrentContractVersion, ActionUploadReset) {
		t.Fatal("expected reset action to be active at current version")
	}
	if mustActionDeprecated(t, NamespaceUpload, "", ActionUploadReset) {
		t.Fatal("expected empty version not to be deprecated")
	}
}

func TestUploadActionCompatibilityRejectsUnknownAction(t *testing.T) {
	if _, err := IsActionContractActive(NamespaceUpload, CurrentContractVersion, Action("does-not-exist")); err == nil {
		t.Fatal("expected unknown action to return an error for active check")
	}
	if _, err := IsActionContractDeprecated(NamespaceUpload, "2026.03.02", Action("does-not-exist")); err == nil {
		t.Fatal("expected unknown action to return an error for deprecated check")
	}
}

func TestActionContractsMatchContractDescriptors(t *testing.T) {
	cases := []struct {
		name      string
		namespace Namespace
		action    Action
		contract  contracts.ContractID
	}{
		{
			name:      "upload open-endpoint",
			namespace: NamespaceUpload,
			action:    ActionUploadOpenEndpoint,
			contract:  contracts.UploadOpenEndpointAction,
		},
		{
			name:      "upload reset",
			namespace: NamespaceUpload,
			action:    ActionUploadReset,
			contract:  contracts.UploadResetAction,
		},
		{
			name:      "c2c connect",
			namespace: NamespaceC2C,
			action:    ActionC2CConnect,
			contract:  contracts.C2CConnectAction,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contract := mustContractForAction(t, tc.namespace, tc.action)
			catalogDescriptor, ok := contracts.ContractDescriptorFor(tc.contract)
			if !ok {
				t.Fatalf("expected contract descriptor %s to be present", tc.contract)
			}
			if contract.Contract.Name != catalogDescriptor.Name {
				t.Fatalf("expected action contract name %q to match catalog name %q", contract.Contract.Name, catalogDescriptor.Name)
			}
			if contract.Contract.Description != catalogDescriptor.Description {
				t.Fatalf("expected action contract description %q to match catalog description %q", contract.Contract.Description, catalogDescriptor.Description)
			}
			if contract.Contract.IntroducedIn != catalogDescriptor.IntroducedIn {
				t.Fatalf("expected action contract introduced version %q to match catalog introduced version %q", contract.Contract.IntroducedIn, catalogDescriptor.IntroducedIn)
			}
			if contract.Contract.Compatibility != catalogDescriptor.Compatibility {
				t.Fatalf("expected action contract compatibility %q to match catalog compatibility %q", contract.Contract.Compatibility, catalogDescriptor.Compatibility)
			}
		})
	}
}

func TestContractBindingsConformanceMatrix(t *testing.T) {
	bindings := contracts.ActionContractBindings()
	if len(bindings) == 0 {
		t.Fatal("expected canonical action contract bindings")
	}

	seen := make(map[contracts.ActionContractBinding]struct{}, len(bindings))
	for _, binding := range bindings {
		namespace := Namespace(binding.Namespace)
		action := Action(binding.Action)

		parsed, err := ParseAction(namespace, string(action))
		if err != nil {
			t.Fatalf("expected parser to accept %s:%s: %v", namespace, action, err)
		}
		if parsed != action {
			t.Fatalf("expected parser output %q, got %q", action, parsed)
		}

		contract := mustContractForAction(t, namespace, action)
		if contract.Contract.Name == "" || contract.Contract.Description == "" {
			t.Fatalf("contract metadata missing descriptor fields for %s:%s", namespace, action)
		}

		if namespace == NamespaceC2C {
			payloadBytes, err := MarshalC2CPayload(action, "HomeWiFi", "secret")
			if err != nil {
				t.Fatalf("marshal c2c payload for %s failed: %v", action, err)
			}
			var payload C2CPayload
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				t.Fatalf("decode c2c payload for %s failed: %v", action, err)
			}
			if payload.Type != C2CPayloadType {
				t.Fatalf("unexpected c2c payload type for %s: %q", action, payload.Type)
			}
			if payload.Payload.Action != string(action) {
				t.Fatalf("payload action mismatch for %s: got %q", action, payload.Payload.Action)
			}
		}

		seen[binding] = struct{}{}
	}

	if len(seen) != len(bindings) {
		t.Fatalf("expected %d unique bindings, got %d", len(bindings), len(seen))
	}
}

func TestContractNamespacesRejectUnsupportedActions(t *testing.T) {
	bindings := contracts.ActionContractBindings()
	if len(bindings) == 0 {
		t.Fatal("expected canonical action contract bindings")
	}

	seenNamespaces := make(map[Namespace]struct{})
	for _, binding := range bindings {
		seenNamespaces[Namespace(binding.Namespace)] = struct{}{}
	}

	for namespace := range seenNamespaces {
		_, err := ParseAction(namespace, "definitely-unsupported-action")
		if err == nil {
			t.Fatalf("expected unsupported action error for namespace %s", namespace)
		}
		var unsupported UnsupportedActionError
		if !errors.As(err, &unsupported) {
			t.Fatalf("expected UnsupportedActionError for namespace %s, got %T: %v", namespace, err, err)
		}
		if unsupported.Namespace != namespace {
			t.Fatalf("unsupported namespace mismatch: got %s want %s", unsupported.Namespace, namespace)
		}
	}
}
