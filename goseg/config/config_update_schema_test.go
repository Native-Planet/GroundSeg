package config

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"groundseg/structs"
)

func TestConfPatchRegistryMatchesPatchStructFields(t *testing.T) {
	fields := reflect.TypeOf(ConfPatch{})
	registered := make(map[string]struct{}, fields.NumField())
	for _, field := range allConfPatchFields() {
		if field.patchField == "" {
			continue
		}
		registered[field.patchField] = struct{}{}
		if _, ok := fields.FieldByName(field.patchField); !ok {
			t.Fatalf("confPatchRegistry references unknown field %q", field.patchField)
		}
	}

	missing := []string{}
	observed := collectConfPatchFieldNames(fields)
	for name := range observed {
		if _, ok := registered[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("confPatchRegistry missing %d fields: %s", len(missing), strings.Join(missing, ", "))
	}
}

func collectConfPatchFieldNames(typ reflect.Type) map[string]struct{} {
	fields := make(map[string]struct{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			for name := range collectConfPatchFieldNames(field.Type) {
				fields[name] = struct{}{}
			}
			continue
		}
		fields[field.Name] = struct{}{}
	}
	return fields
}

func TestBuildConfPatchByKeyRejectsDuplicateKeys(t *testing.T) {
	_, err := buildConfPatchByKey([]confPatchField{
		{key: "duplicate", patchField: "Piers"},
		{key: "duplicate", patchField: "WgOn"},
	})
	if err == nil {
		t.Fatalf("expected duplicate config patch key error")
	}
}

func TestBuildConfigPatchSupportsKnownAndUnsupportedKeys(t *testing.T) {
	if _, err := buildConfigPatch(map[string]interface{}{
		"setup":                  "startram",
		"isEMMCMachine":          true,
		"piers":                  []string{"desk"},
		"startramSetReminderOne": true,
	}); err == nil || !strings.Contains(err.Error(), "unsupported config key: isEMMCMachine") {
		t.Fatalf("expected unsupported key error for isEMMCMachine, got %v", err)
	}
}

func TestApplyConfPatchMergesAuthorizedSessions(t *testing.T) {
	initial := structs.SysConfig{}
	initial.Sessions = structs.AuthSessionBag{
		Authorized: map[string]structs.SessionInfo{
			"existing": {Hash: "existing"},
		},
		Unauthorized: nil,
	}
	patch := &ConfPatch{
		AuthSessionPatch: AuthSessionPatch{
			AuthorizedSessions: map[string]structs.SessionInfo{
				"added": {Hash: "added"},
			},
		},
	}
	applyConfPatch(&initial, patch)
	if len(initial.Sessions.Authorized) != 2 {
		t.Fatalf("expected merged authorized sessions, got %+v", initial.Sessions.Authorized)
	}
}
