package structs

import (
	"encoding/json"
	"testing"
)

func TestSubdomainUnmarshalParsesNumericPort(t *testing.T) {
	var sub Subdomain
	err := json.Unmarshal([]byte(`{"alias":"api","port":8443,"status":"ok","svc_type":"urbit","url":"api.example"}`), &sub)
	if err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}
	if sub.Port != 8443 {
		t.Fatalf("expected parsed port 8443, got %d", sub.Port)
	}
	if sub.Alias != "api" || sub.SvcType != "urbit" {
		t.Fatalf("unexpected subdomain fields: %+v", sub)
	}
}

func TestSubdomainUnmarshalDefaultsPortForNonNumericType(t *testing.T) {
	var sub Subdomain
	err := json.Unmarshal([]byte(`{"alias":"api","port":"8443"}`), &sub)
	if err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}
	if sub.Port != 0 {
		t.Fatalf("expected non-numeric port to default to 0, got %d", sub.Port)
	}
}

func TestSubdomainUnmarshalReturnsErrorForInvalidJSON(t *testing.T) {
	var sub Subdomain
	if err := json.Unmarshal([]byte("{"), &sub); err == nil {
		t.Fatal("expected invalid json unmarshal error")
	}
}

func TestStartramSvcRespUnmarshalHandlesStringLease(t *testing.T) {
	var resp StartramSvcResp
	err := json.Unmarshal([]byte(`{"action":"create","lease":"2027-01-01","subdomain":"ship","status":"ok"}`), &resp)
	if err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}
	if resp.Lease != "2027-01-01" {
		t.Fatalf("expected lease to be preserved, got %q", resp.Lease)
	}
	if resp.Action != "create" || resp.Subdomain != "ship" {
		t.Fatalf("unexpected response fields: %+v", resp)
	}
}

func TestStartramSvcRespUnmarshalDefaultsLeaseForNonStringType(t *testing.T) {
	var resp StartramSvcResp
	err := json.Unmarshal([]byte(`{"lease":12345,"action":"create"}`), &resp)
	if err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}
	if resp.Lease != "" {
		t.Fatalf("expected non-string lease to default empty string, got %q", resp.Lease)
	}
}

func TestStartramSvcRespUnmarshalReturnsErrorForInvalidJSON(t *testing.T) {
	var resp StartramSvcResp
	if err := json.Unmarshal([]byte("{"), &resp); err == nil {
		t.Fatal("expected invalid json unmarshal error")
	}
}
