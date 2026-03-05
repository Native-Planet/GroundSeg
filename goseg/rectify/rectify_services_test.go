package rectify

import (
	"fmt"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestReconcilePatpStatePropagatesServiceCreationErrors(t *testing.T) {
	reconciler := &StartramRetrieveReconciler{
		createServiceFn: func(subdomain, svcType string) error {
			return fmt.Errorf("service create failed for %s (%s)", subdomain, svcType)
		},
	}

	plan, err := reconciler.reconcilePatpState(
		"zod",
		structs.UrbitDocker{},
		nil,
		config.StartramSettings{EndpointURL: "api.startram.io"},
	)
	if err == nil {
		t.Fatal("expected service creation error to be returned")
	}
	if plan.serviceCreated {
		t.Fatal("expected serviceCreated to be false when service creation fails")
	}
	if !strings.Contains(err.Error(), "create urbit service zod") {
		t.Fatalf("expected urbit service creation context in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "create minio service s3.zod") {
		t.Fatalf("expected minio service creation context in error, got %v", err)
	}
}

func TestReconcilePatpStateCreatesMissingServices(t *testing.T) {
	var calls []string
	reconciler := &StartramRetrieveReconciler{
		createServiceFn: func(subdomain, svcType string) error {
			calls = append(calls, subdomain+":"+svcType)
			return nil
		},
	}

	plan, err := reconciler.reconcilePatpState(
		"zod",
		structs.UrbitDocker{},
		nil,
		config.StartramSettings{EndpointURL: "api.startram.io"},
	)
	if err != nil {
		t.Fatalf("expected service creation success, got %v", err)
	}
	if !plan.serviceCreated {
		t.Fatal("expected serviceCreated to remain true after successful creation")
	}
	if len(calls) != 2 {
		t.Fatalf("expected two service creation calls, got %d", len(calls))
	}
	if calls[0] != "zod:urbit" {
		t.Fatalf("unexpected first service call: %s", calls[0])
	}
	if calls[1] != "s3.zod:minio" {
		t.Fatalf("unexpected second service call: %s", calls[1])
	}
}
