package config

import (
	"strings"
	"testing"

	"groundseg/structs"
)

func TestConfPatchHasUpdatesRecognizesConfiguredFields(t *testing.T) {
	empty := &ConfPatch{}
	if empty.hasUpdates() {
		t.Fatal("expected empty patch to report no updates")
	}

	withUpdate := &ConfPatch{
		ConnectivityPatch: ConnectivityPatch{
			WgOn: ptrBool(true),
		},
	}
	if !withUpdate.hasUpdates() {
		t.Fatal("expected non-empty patch to report updates")
	}
}

func TestBuildConfigPatchParsesKnownFields(t *testing.T) {
	patch, err := buildConfigPatch(map[string]interface{}{
		"wgOn":     true,
		"setup":    "ready",
		"piers":    []interface{}{"~zod"},
		"snapTime": 12,
	})
	if err != nil {
		t.Fatalf("expected known fields to parse successfully: %v", err)
	}
	if patch.WgOn == nil || !*patch.WgOn {
		t.Fatalf("expected WgOn patch to be set true, got %#v", patch.WgOn)
	}
	if patch.Setup == nil || *patch.Setup != "ready" {
		t.Fatalf("expected setup patch to be set to ready, got %#v", patch.Setup)
	}
	if patch.Piers == nil || len(*patch.Piers) != 1 || (*patch.Piers)[0] != "~zod" {
		t.Fatalf("expected piers patch to be parsed, got %#v", patch.Piers)
	}
	if patch.SnapTime == nil || *patch.SnapTime != 12 {
		t.Fatalf("expected snap time patch to be parsed, got %#v", patch.SnapTime)
	}
}

func TestParseStringSliceValueRejectsMixedTypes(t *testing.T) {
	_, err := parseStringSliceValue("piers", []interface{}{"~nec", 7})
	if err == nil || !strings.Contains(err.Error(), "invalid piers item 1 value: int") {
		t.Fatalf("expected typed item parse error, got %v", err)
	}
}

func TestParseSessionMapCopiesSessionEntries(t *testing.T) {
	input := map[string]structs.SessionInfo{
		"token": {Hash: "abc"},
	}
	parsed, err := parseSessionMap("sessions", input)
	if err != nil {
		t.Fatalf("expected session map parse to succeed: %v", err)
	}
	input["token"] = structs.SessionInfo{Hash: "mutated"}
	if parsed["token"].Hash != "abc" {
		t.Fatalf("expected parsed sessions to be copied, got %#v", parsed["token"])
	}
}

func TestCopyDiskWarningsReturnsDetachedCopy(t *testing.T) {
	source := map[string]structs.DiskWarning{
		"disk0": {Eighty: true},
	}
	copied := copyDiskWarnings(source)
	if len(copied) != 1 || !copied["disk0"].Eighty {
		t.Fatalf("unexpected copied disk warnings: %#v", copied)
	}
	source["disk0"] = structs.DiskWarning{Ninety: true}
	if copied["disk0"].Ninety {
		t.Fatalf("expected copied warning map to be detached from source mutations")
	}
}

func ptrBool(v bool) *bool {
	return &v
}
