package contracts

import "testing"

func TestContractMetadataForKnownIDs(t *testing.T) {
	cases := []ContractID{
		UploadOpenEndpointAction,
		UploadResetAction,
		C2CConnectAction,
		APIConnectionError,
	}
	for _, id := range cases {
		t.Run(string(id), func(t *testing.T) {
			metadata := contractMetadataFor(id)
			if metadata.IntroducedIn == "" {
				t.Fatalf("expected introduced version for %s", id)
			}
			if metadata.Compatibility == "" {
				t.Fatalf("expected compatibility for %s", id)
			}
		})
	}
}

func TestContractMetadataForPanicsOnUnknownID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unknown contract metadata id")
		}
	}()
	_ = contractMetadataFor("unknown.contract.id")
}
