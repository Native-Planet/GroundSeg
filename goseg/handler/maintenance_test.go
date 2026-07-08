package handler

import "testing"

func TestBeginShipMaintenanceRejectsConcurrentMaintenance(t *testing.T) {
	shipMaintenance.Lock()
	shipMaintenance.active = make(map[string]string)
	shipMaintenance.Unlock()

	done, err := beginShipMaintenance("zod", "pack")
	if err != nil {
		t.Fatalf("beginShipMaintenance returned error: %v", err)
	}

	if _, err := beginShipMaintenance("zod", "chop"); err == nil {
		t.Fatal("expected concurrent maintenance to be rejected")
	}

	done()
	done, err = beginShipMaintenance("zod", "chop")
	if err != nil {
		t.Fatalf("beginShipMaintenance after release returned error: %v", err)
	}
	done()
}
