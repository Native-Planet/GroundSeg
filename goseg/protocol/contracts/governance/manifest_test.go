package governance

import "testing"

func TestActionDeclarationsHaveUniqueBindings(t *testing.T) {
	type bindingKey struct {
		namespace string
		action    string
	}
	seen := make(map[bindingKey]struct{})
	for _, declaration := range ActionDeclarations() {
		key := bindingKey{namespace: declaration.Namespace, action: declaration.Action}
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate action declaration for %s:%s", declaration.Namespace, declaration.Action)
		}
		seen[key] = struct{}{}
		if declaration.ContractID == "" {
			t.Fatalf("missing contract id for %s:%s", declaration.Namespace, declaration.Action)
		}
		if declaration.Owner == "" {
			t.Fatalf("missing owner for %s:%s", declaration.Namespace, declaration.Action)
		}
	}
}

func TestUploadActionPayloadRulesAreDisjoint(t *testing.T) {
	uploadActions := UploadActionDeclarations()
	if len(uploadActions) == 0 {
		t.Fatal("expected upload declarations")
	}
	for _, declaration := range uploadActions {
		if declaration.RequiredPayloads == 0 {
			t.Fatalf("upload action %s missing required payload rule", declaration.Action)
		}
		if declaration.RequiredPayloads.Has(declaration.ForbiddenPayloads) {
			t.Fatalf("upload action %s has overlapping required and forbidden payload rules", declaration.Action)
		}
	}
}

func TestContractDeclarationsHaveUniqueIDs(t *testing.T) {
	seen := make(map[string]struct{})
	for _, declaration := range StartramContractDeclarations() {
		if declaration.ID == "" {
			t.Fatal("expected non-empty contract id")
		}
		if _, exists := seen[declaration.ID]; exists {
			t.Fatalf("duplicate contract id declaration: %s", declaration.ID)
		}
		seen[declaration.ID] = struct{}{}
	}
}
