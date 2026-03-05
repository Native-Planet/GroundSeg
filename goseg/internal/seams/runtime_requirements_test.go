package seams

import (
	"strings"
	"testing"
)

type taggedCallbacks struct {
	RequiredFn func() `runtime:"group-a" runtime_name:"required-fn"`
	OtherFn    func() `runtime:"group-b" runtime_name:"other-fn"`
	Untagged   func()
}

type nestedTaggedCallbacks struct {
	DeepFn func() `runtime:"group-a" runtime_name:"deep-fn"`
}

type embeddedTaggedCallbacks struct {
	taggedCallbacks
	nestedTaggedCallbacks
}

func TestMissingTaggedCallbacksRespectsGroupFilterAndEmbeddedFields(t *testing.T) {
	subject := embeddedTaggedCallbacks{}
	subject.RequiredFn = func() {}
	requiredGroups := map[string]struct{}{"group-a": {}}

	missing := MissingTaggedCallbacks(subject, requiredGroups)
	if len(missing) != 1 || missing[0] != "deep-fn" {
		t.Fatalf("expected only deep-fn missing for group-a, got %v", missing)
	}
}

func TestValidateCallbacksReturnsReadableError(t *testing.T) {
	requirements := NewCallbackRequirementsWithGroups("group-a")
	err := requirements.ValidateCallbacks(taggedCallbacks{}, "runtime-under-test")
	if err == nil {
		t.Fatal("expected missing callback validation error")
	}
	if !strings.Contains(err.Error(), "runtime-under-test missing required callbacks: required-fn") {
		t.Fatalf("unexpected validation error message: %v", err)
	}
}
