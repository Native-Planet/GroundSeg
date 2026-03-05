package conformance

import (
	"fmt"

	"groundseg/protocol/contracts"
)

type ActionContractFixture struct {
	Namespace contracts.ActionNamespace
	Action    contracts.ActionVerb
	Contract  contracts.ContractID
}

func ActionContractFixtures() []ActionContractFixture {
	bindings := contracts.ActionContractBindings()
	fixtures := make([]ActionContractFixture, 0, len(bindings))
	for _, binding := range bindings {
		fixtures = append(fixtures, ActionContractFixture{
			Namespace: binding.Namespace,
			Action:    binding.Action,
			Contract:  binding.Contract,
		})
	}
	return fixtures
}

func ValidateActionContractFixture(fixture ActionContractFixture) error {
	_, ok := contracts.ActionContractFor(
		fixture.Namespace,
		fixture.Action,
	)
	if !ok {
		return fmt.Errorf("missing canonical contract descriptor %s:%s", fixture.Namespace, fixture.Action)
	}
	canonicalBinding := contracts.ActionContractBinding{}
	bindingFound := false
	for _, binding := range contracts.ActionContractBindings() {
		if binding.Namespace == fixture.Namespace &&
			binding.Action == fixture.Action {
			canonicalBinding = binding
			bindingFound = true
			break
		}
	}
	if !bindingFound {
		return fmt.Errorf("missing canonical action binding %s:%s", fixture.Namespace, fixture.Action)
	}
	if canonicalBinding.Contract != fixture.Contract {
		return fmt.Errorf("contract mismatch for %s:%s: got %s want %s", fixture.Namespace, fixture.Action, canonicalBinding.Contract, fixture.Contract)
	}
	descriptor, ok := contracts.ActionContractFor(fixture.Namespace, fixture.Action)
	if !ok {
		return fmt.Errorf("missing canonical descriptor for %s:%s", fixture.Namespace, fixture.Action)
	}
	if descriptor.Name == "" {
		return fmt.Errorf("contract descriptor missing name for %s:%s", fixture.Namespace, fixture.Action)
	}
	return nil
}

func ValidateFixturesForNamespace(namespace contracts.ActionNamespace) (int, error) {
	count := 0
	for _, fixture := range ActionContractFixtures() {
		if fixture.Namespace != namespace {
			continue
		}
		if err := ValidateActionContractFixture(fixture); err != nil {
			return count, fmt.Errorf("validate fixture %s:%s: %w", fixture.Namespace, fixture.Action, err)
		}
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("no fixtures registered for namespace %s", namespace)
	}
	return count, nil
}
