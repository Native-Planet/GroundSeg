package contracts

import (
	"testing"
	"time"
)

func TestContractCatalogHasLifecycleMetadata(t *testing.T) {
	for id, descriptor := range contractCatalog {
		if descriptor.Name == "" {
			t.Fatalf("contract %s has empty name", id)
		}
		if descriptor.Description == "" {
			t.Fatalf("contract %s has empty description", id)
		}
		if descriptor.Compatibility == "" {
			t.Fatalf("contract %s has empty compatibility", descriptor.Name)
		}
		if descriptor.IntroducedIn == "" {
			t.Fatalf("contract %s has empty introduction version", descriptor.Name)
		}
		if _, err := time.Parse(contractVersionLayout, descriptor.IntroducedIn); err != nil {
			t.Fatalf("contract %s has invalid introduced version %q: %v", descriptor.Name, descriptor.IntroducedIn, err)
		}
		if descriptor.DeprecatedIn != "" {
			if _, err := time.Parse(contractVersionLayout, descriptor.DeprecatedIn); err != nil {
				t.Fatalf("contract %s has invalid deprecated version %q: %v", descriptor.Name, descriptor.DeprecatedIn, err)
			}
		}
		if descriptor.RemovedIn != "" {
			if _, err := time.Parse(contractVersionLayout, descriptor.RemovedIn); err != nil {
				t.Fatalf("contract %s has invalid removed version %q: %v", descriptor.Name, descriptor.RemovedIn, err)
			}
		}
		if descriptor.RemovedIn != "" && descriptor.DeprecatedIn != "" && !IsVersionAtLeastOrEqual(descriptor.RemovedIn, descriptor.DeprecatedIn) {
			t.Fatalf("contract %s has removed-before-deprecated lifecycle windows", descriptor.Name)
		}
		if !descriptor.IsActive(CurrentContractVersion) && descriptor.RemovedIn == "" {
			t.Fatalf("contract %s is currently inactive without removal date", descriptor.Name)
		}
	}
}

func TestContractDescriptorForUnknownIDReturnsMissing(t *testing.T) {
	if descriptor, ok := ContractDescriptorFor("protocol.contracts.does-not-exist"); ok {
		t.Fatalf("expected unknown contract id lookup to fail, got %+v", descriptor)
	}
}

func TestMustContractDescriptorPanicsForUnknownID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustContractDescriptor to panic for unknown contract id")
		}
	}()
	_ = MustContractDescriptor(ContractID("protocol.contracts.does-not-exist"))
}

func TestActionContractBindingsHaveActiveDescriptors(t *testing.T) {
	for _, binding := range ActionContractBindings {
		if string(binding.Namespace) == "" {
			t.Fatalf("missing namespace for action binding %q", binding.Action)
		}
		if binding.Action == "" {
			t.Fatalf("missing action value for namespace %s", binding.Namespace)
		}
		descriptor, ok := ContractDescriptorFor(binding.Contract)
		if !ok {
			t.Fatalf("missing contract %q for action %s:%s", binding.Contract, binding.Namespace, binding.Action)
		}
		if descriptor.Name == "" {
			t.Fatalf("contract %q has empty name", binding.Contract)
		}
		if descriptor.Description == "" {
			t.Fatalf("contract %q has empty description", binding.Contract)
		}
	}
}

func TestActionContractBindingsAreDeterministicallyOrdered(t *testing.T) {
	uploadBindings := ActionContractBindingsForNamespace(string(ActionNamespaceUpload))
	if len(uploadBindings) != 2 {
		t.Fatalf("expected 2 upload action bindings, got %d", len(uploadBindings))
	}
	if uploadBindings[0].Action != "open-endpoint" || uploadBindings[1].Action != "reset" {
		t.Fatalf("unexpected upload binding ordering: %#v", uploadBindings)
	}
	c2cBindings := ActionContractBindingsForNamespace(string(ActionNamespaceC2C))
	if len(c2cBindings) != 1 || c2cBindings[0].Action != "connect" {
		t.Fatalf("unexpected c2c binding ordering: %#v", c2cBindings)
	}
}
