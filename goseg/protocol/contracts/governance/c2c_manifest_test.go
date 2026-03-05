package governance

import (
	"testing"

	"groundseg/protocol/contracts/catalog/common"
)

func TestC2CActionDeclarationsExposeConnectContract(t *testing.T) {
	declarations := c2cActionDeclarations()
	if len(declarations) != 1 {
		t.Fatalf("expected one c2c declaration, got %d", len(declarations))
	}

	declaration := declarations[0]
	if declaration.Namespace != NamespaceC2C {
		t.Fatalf("unexpected c2c namespace: %s", declaration.Namespace)
	}
	if declaration.Action != ActionC2CConnect {
		t.Fatalf("unexpected c2c action token: %s", declaration.Action)
	}
	if declaration.ContractID != C2CConnectContractID {
		t.Fatalf("unexpected c2c contract id: %s", declaration.ContractID)
	}
	if declaration.Owner != string(common.OwnerSystemWiFi) {
		t.Fatalf("unexpected c2c owner: %s", declaration.Owner)
	}
	if declaration.RequiredPayloads != 0 || declaration.ForbiddenPayloads != 0 {
		t.Fatalf("expected c2c action to have no upload payload policy flags, got required=%d forbidden=%d", declaration.RequiredPayloads, declaration.ForbiddenPayloads)
	}
}
