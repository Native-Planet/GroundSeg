package tokens

import (
	"net/http"
	"testing"
)

func TestRequestIdentityFromRequestRequiresRequest(t *testing.T) {
	ip, userAgent, err := requestIdentityFromRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
	if ip != "" || userAgent != "" {
		t.Fatalf("expected empty identity for nil request, got ip=%q userAgent=%q", ip, userAgent)
	}
}

func TestRequestIdentityFromRequestReadsForwardedAddress(t *testing.T) {
	req := &http.Request{
		Header: map[string][]string{
			"X-Forwarded-For": {"1.2.3.4, 10.0.0.1"},
			"User-Agent":      {"unit-test-agent"},
		},
	}
	ip, userAgent, err := requestIdentityFromRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "1.2.3.4" {
		t.Fatalf("unexpected ip: %q", ip)
	}
	if userAgent != "unit-test-agent" {
		t.Fatalf("unexpected user-agent: %q", userAgent)
	}
}

func TestCreateTokenRequiresRequest(t *testing.T) {
	token, err := CreateToken(nil, nil, false)
	if err == nil {
		t.Fatalf("expected CreateToken to error on nil request, got token=%v", token)
	}
}
