package actions

import (
	"errors"
	"groundseg/protocol/contracts"
	"reflect"
	"time"
	"testing"
)

func versionBefore(reference string) string {
	referenceTime, err := time.Parse("2006.01.02", reference)
	if err != nil {
		return reference
	}
	return referenceTime.Add(-24 * time.Hour).Format("2006.01.02")
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
	_, err := ParseUploadAction("definitely-not-upload")
	if err == nil {
		t.Fatal("expected unknown upload action to be rejected")
	}
	var unsupported UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
	}
}

func TestSupportedActionsUnknownNamespaceReturnsNil(t *testing.T) {
	if got := SupportedActions("invalid"); got != nil {
		t.Fatalf("expected nil supported actions for invalid namespace, got %v", got)
	}
}

func TestActionMetasReturnsContractsByNamespace(t *testing.T) {
	upload := ActionMetas(NamespaceUpload)
	expectedUpload := []ActionMeta{
		{Action: ActionUploadOpenEndpoint, Description: "open upload endpoint"},
		{Action: ActionUploadReset, Description: "reset upload session"},
	}
	if !reflect.DeepEqual(upload, expectedUpload) {
		t.Fatalf("unexpected upload action metadata: got %#v", upload)
	}

	c2c := ActionMetas(NamespaceC2C)
	expectedC2C := []ActionMeta{
		{Action: ActionC2CConnect, Description: "connect c2c client"},
	}
	if !reflect.DeepEqual(c2c, expectedC2C) {
		t.Fatalf("unexpected c2c action metadata: got %#v", c2c)
	}
}

func TestActionMetasForInvalidNamespaceReturnsNil(t *testing.T) {
	if got := ActionMetas("invalid"); got != nil {
		t.Fatalf("expected nil action metadata for invalid namespace, got %v", got)
	}
}

func TestActionMetasReturnsCopyByValue(t *testing.T) {
	first := ActionMetas(NamespaceUpload)
	if len(first) == 0 {
		t.Fatal("expected at least one upload action metadata entry")
	}
	if first[0].Action != ActionUploadOpenEndpoint {
		t.Fatalf("precondition failed: first upload action %q", first[0].Action)
	}
	first[0].Action = "mutated"
	first[0].Description = "mutated description"

	second := ActionMetas(NamespaceUpload)
	if second[0].Action == first[0].Action {
		t.Fatalf("expected ActionMetas return value to be copied by value, got mutated action %q", second[0].Action)
	}
	if second[0].Description == "mutated description" {
		t.Fatalf("expected ActionMetas return value to be insulated from external mutation")
	}
}

func TestParseUploadActionRejectsUnknown(t *testing.T) {
	_, err := ParseUploadAction("bogus")
	if err == nil {
		t.Fatalf("expected unknown upload action to be rejected")
	}
	unsupported, ok := err.(UnsupportedActionError)
	if !ok {
		t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
	}
	if unsupported.Namespace != NamespaceUpload {
		t.Fatalf("expected upload namespace, got %s", unsupported.Namespace)
	}
}

func TestSupportedUploadActionsMatchesContract(t *testing.T) {
	got := SupportedUploadActions()
	expected := []Action{ActionUploadOpenEndpoint, ActionUploadReset}
	if len(got) != len(expected) {
		t.Fatalf("expected %d upload actions, got %d", len(expected), len(got))
	}
	for i, action := range expected {
		if got[i] != action {
			t.Fatalf("expected upload action %q at index %d, got %q", action, i, got[i])
		}
		if _, err := ParseUploadAction(string(action)); err != nil {
			t.Fatalf("expected action %q to parse as supported: %v", action, err)
		}
	}
}

func TestUploadActionContractsIncludesExpectedActions(t *testing.T) {
	got := UploadActionContracts()
	if len(got) != 2 {
		t.Fatalf("expected 2 upload action contracts, got %d", len(got))
	}

	got[0].Action = "mutated"
	gotCopy := UploadActionContracts()
	if gotCopy[0].Action == "mutated" {
		t.Fatalf("expected upload action contracts to be copied by value")
	}
}

func TestUploadActionContractsAreOrdered(t *testing.T) {
	got := UploadActionContracts()
	want := []UploadActionContract{
		{
			ActionContract: ActionContract{
				Action:      ActionUploadOpenEndpoint,
				Description: "open upload endpoint",
				Contract: contracts.ContractDescriptor{
					Description: "open upload endpoint",
					Name:        "UploadActionOpenEndpoint",
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
				Description: "reset upload session",
				Contract: contracts.ContractDescriptor{
					Description: "reset upload session",
					Name:        "UploadActionReset",
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
			contract, ok := UploadActionContractForAction(action)
			if !ok {
				t.Fatalf("expected %q contract to resolve", action)
			}
			if contract.Action != action {
				t.Fatalf("unexpected contract action: %v", contract.Action)
			}
		})
	}
}

func TestUploadActionContractForActionUnknownAction(t *testing.T) {
	_, ok := UploadActionContractForAction("upload-action-does-not-exist")
	if ok {
		t.Fatal("expected unknown upload action contract lookup to miss")
	}
}

func TestUploadActionContractForActionReturnsCopy(t *testing.T) {
	got := UploadActionContracts()
	if len(got) == 0 {
		t.Fatal("expected at least one upload action contract")
	}
	contract := got[0]
	contract.Action = "mutated"

	rechecked, ok := UploadActionContractForAction(got[0].Action)
	if !ok {
		t.Fatal("expected looked-up action to resolve")
	}
	if rechecked.Action != got[0].Action {
		t.Fatalf("expected contract copy lookup to stay stable, got %q", rechecked.Action)
	}
}

func TestSupportedC2CActionsMatchesContract(t *testing.T) {
	got := SupportedC2CActions()
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
		meta, ok := UploadActionContractMetadataForAction(action)
		if !ok {
			t.Fatalf("missing metadata for action %q", action)
		}
		if meta.IntroducedIn == "" {
			t.Fatalf("expected introduced version for %q", action)
		}
		if meta.Compatibility == "" {
			t.Fatalf("expected compatibility for %q", action)
		}
	}
}

func TestC2CActionMetadataContainsLifecycle(t *testing.T) {
	meta, ok := C2CActionContractMetadataForAction(ActionC2CConnect)
	if !ok {
		t.Fatal("missing metadata for C2C connect action")
	}
	if meta.IntroducedIn == "" {
		t.Fatal("expected introduced version for C2C connect action")
	}
	if meta.Compatibility == "" {
		t.Fatal("expected compatibility for C2C connect action")
	}
}

func TestUploadActionCompatibilityChecksUseMetadata(t *testing.T) {
	if !IsUploadActionContractActive(CurrentContractVersion, ActionUploadOpenEndpoint) {
		t.Fatal("expected upload open-endpoint action to be active at current version")
	}
	if IsUploadActionContractDeprecated(CurrentContractVersion, ActionUploadOpenEndpoint) {
		t.Fatal("did not expect action to be deprecated at current version")
	}
	if IsUploadActionContractActive("2026.01.01", ActionUploadOpenEndpoint) {
		t.Fatal("did not expect action to be active before introduction date")
	}
}

func TestC2CActionContractForActionResolvesKnownAction(t *testing.T) {
	contract, ok := C2CActionContractForAction(ActionC2CConnect)
	if !ok {
		t.Fatalf("expected C2C connect action contract to resolve")
	}
	if contract.Action != ActionC2CConnect {
		t.Fatalf("unexpected C2C contract action: %q", contract.Action)
	}
}

func TestC2CActionCompatibilityChecksUseMetadata(t *testing.T) {
	if !IsC2CActionContractActive(CurrentContractVersion, ActionC2CConnect) {
		t.Fatal("expected C2C connect action to be active at current version")
	}
	if IsC2CActionContractDeprecated(CurrentContractVersion, ActionC2CConnect) {
		t.Fatal("did not expect action to be deprecated at current version")
	}
	if IsC2CActionContractActive("2026.01.01", ActionC2CConnect) {
		t.Fatal("did not expect C2C connect action to be active before introduction date")
	}
}

func TestUploadActionCompatibilityBoundaryCases(t *testing.T) {
	if IsUploadActionContractActive("", ActionUploadOpenEndpoint) {
		t.Fatal("expected empty version to be inactive")
	}
	if IsUploadActionContractActive("2026.01.01", ActionUploadOpenEndpoint) {
		t.Fatal("expected open-endpoint to be inactive before introduction")
	}
	if IsUploadActionContractActive("bad-version", ActionUploadOpenEndpoint) {
		t.Fatal("expected malformed version to be inactive")
	}
	if IsUploadActionContractDeprecated(CurrentContractVersion, ActionUploadOpenEndpoint) {
		t.Fatal("expected current action not to be deprecated")
	}
	if IsUploadActionContractDeprecated("", ActionUploadOpenEndpoint) {
		t.Fatal("expected empty version not to be deprecated")
	}
	if !IsUploadActionContractActive(CurrentContractVersion, ActionUploadReset) {
		t.Fatal("expected reset action to be active at current version")
	}
	if IsUploadActionContractDeprecated("", ActionUploadReset) {
		t.Fatal("expected empty version not to be deprecated")
	}
}

func TestUploadActionCompatibilityRejectsUnknownAction(t *testing.T) {
	if IsUploadActionContractActive(CurrentContractVersion, Action("does-not-exist")) {
		t.Fatal("expected unknown action to be inactive")
	}
	if IsUploadActionContractDeprecated("2026.03.02", Action("does-not-exist")) {
		t.Fatal("expected unknown action to be non-deprecated")
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
			contract:  contracts.UploadActionOpenEndpoint,
		},
		{
			name:      "upload reset",
			namespace: NamespaceUpload,
			action:    ActionUploadReset,
			contract:  contracts.UploadActionReset,
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
			var contract ActionContract
			var ok bool
			switch tc.namespace {
			case NamespaceUpload:
				uploadContract, found := UploadActionContractForAction(tc.action)
				ok = found
				contract = uploadContract.ActionContract
			case NamespaceC2C:
				contract, ok = C2CActionContractForAction(tc.action)
			default:
				t.Fatalf("unsupported namespace: %s", tc.namespace)
			}
			if !ok {
				t.Fatalf("expected action %s in namespace %s to resolve", tc.action, tc.namespace)
			}
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
