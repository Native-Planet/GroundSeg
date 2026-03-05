package familycatalog

import "testing"

func TestProtocolActionSpecs(t *testing.T) {
	specs := ProtocolActionSpecs()
	if len(specs) != 1 {
		t.Fatalf("expected one protocol action spec, got %d", len(specs))
	}
	spec := specs[0]
	if spec.Namespace != NamespaceC2C {
		t.Fatalf("unexpected namespace: %q", spec.Namespace)
	}
	if spec.Action != ActionC2CConnect {
		t.Fatalf("unexpected action: %q", spec.Action)
	}
	if spec.ContractID != C2CConnectContractID {
		t.Fatalf("unexpected contract id: %q", spec.ContractID)
	}
}

func TestUploadActionSpecsExposePayloadRules(t *testing.T) {
	specs := UploadActionSpecs()
	if len(specs) != 2 {
		t.Fatalf("expected two upload action specs, got %d", len(specs))
	}
	seen := map[string]UploadActionSpec{}
	for _, spec := range specs {
		seen[spec.Action] = spec
		if spec.RequiredPayloads == 0 {
			t.Fatalf("required payloads must be set for %s", spec.Action)
		}
		if spec.RequiredPayloads&spec.ForbiddenPayloads != 0 {
			t.Fatalf("required and forbidden payloads overlap for %s", spec.Action)
		}
	}
	openSpec, ok := seen[ActionUploadOpenEndpoint]
	if !ok {
		t.Fatalf("missing upload action spec for %s", ActionUploadOpenEndpoint)
	}
	if openSpec.RequiredPayloads != UploadPayloadRuleOpenEndpoint || openSpec.ForbiddenPayloads != UploadPayloadRuleReset {
		t.Fatalf("unexpected payload rules for %s", ActionUploadOpenEndpoint)
	}
	resetSpec, ok := seen[ActionUploadReset]
	if !ok {
		t.Fatalf("missing upload action spec for %s", ActionUploadReset)
	}
	if resetSpec.RequiredPayloads != UploadPayloadRuleReset || resetSpec.ForbiddenPayloads != UploadPayloadRuleOpenEndpoint {
		t.Fatalf("unexpected payload rules for %s", ActionUploadReset)
	}
}

func TestAllActionSpecsContainUploadAndProtocolEntries(t *testing.T) {
	specs := AllActionSpecs()
	if len(specs) != len(UploadActionSpecs())+len(ProtocolActionSpecs()) {
		t.Fatalf("unexpected total action specs count: %d", len(specs))
	}
	seenContracts := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if _, exists := seenContracts[spec.ContractID]; exists {
			t.Fatalf("duplicate contract id %q", spec.ContractID)
		}
		seenContracts[spec.ContractID] = struct{}{}
	}
	if _, ok := seenContracts[UploadOpenEndpointContractID]; !ok {
		t.Fatalf("missing contract id %s", UploadOpenEndpointContractID)
	}
	if _, ok := seenContracts[UploadResetContractID]; !ok {
		t.Fatalf("missing contract id %s", UploadResetContractID)
	}
	if _, ok := seenContracts[C2CConnectContractID]; !ok {
		t.Fatalf("missing contract id %s", C2CConnectContractID)
	}
}

func TestStartramContractSpecs(t *testing.T) {
	specs := StartramContractSpecs()
	if len(specs) != 1 {
		t.Fatalf("expected one startram contract spec, got %d", len(specs))
	}
	spec := specs[0]
	if spec.ID != StartramAPIConnectionErrorID {
		t.Fatalf("unexpected id: %q", spec.ID)
	}
	if spec.Owner != string(OwnerStartram) {
		t.Fatalf("unexpected owner: %q", spec.Owner)
	}
}
