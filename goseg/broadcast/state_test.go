package broadcast

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"groundseg/structs"
)

func TestCloneBroadcastStateDeepCopiesMutableFields(t *testing.T) {
	input := structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"~zod": {},
		},
	}
	urbit := input.Urbits["~zod"]
	urbit.Info.RemoteTlonBackups = []structs.BackupObject{{Timestamp: 1}}
	urbit.Info.LocalDailyTlonBackups = []structs.BackupObject{{Timestamp: 2}}
	urbit.Info.LocalWeeklyTlonBackups = []structs.BackupObject{{Timestamp: 3}}
	urbit.Info.LocalMonthlyTlonBackups = []structs.BackupObject{{Timestamp: 4}}
	input.Urbits["~zod"] = urbit
	input.System.Info.Drives = map[string]structs.SystemDrive{"sda": {DriveID: 1}}
	input.System.Info.SMART = map[string]bool{"sda": true}
	input.System.Info.Usage.Disk = map[string][2]uint64{"sda": [2]uint64{100, 50}}
	input.System.Info.Usage.RAM = []uint64{100, 200}
	input.System.Info.Wifi.Networks = []string{"test-net"}
	input.System.Transition.Error = []string{"first"}
	input.Profile.Startram.Info.Regions = map[string]structs.StartramRegion{"us-east": {}}
	input.Profile.Startram.Info.StartramServices = []string{"~zod"}
	input.Apps.Penpai.Info.Models = []string{"model-a"}
	input.Logs.Containers.Wireguard.Logs = []any{"wireguard-log"}
	input.Logs.System.Logs = []any{"system-log"}

	cloned := cloneBroadcastState(input)

	clonedUrbit := cloned.Urbits["~zod"]
	clonedUrbit.Info.RemoteTlonBackups[0].Timestamp = 10
	clonedUrbit.Info.LocalDailyTlonBackups[0].Timestamp = 20
	clonedUrbit.Info.LocalWeeklyTlonBackups[0].Timestamp = 30
	clonedUrbit.Info.LocalMonthlyTlonBackups[0].Timestamp = 40
	cloned.Urbits["~zod"] = clonedUrbit
	cloned.System.Info.Usage.RAM[0] = 999
	cloned.System.Info.Wifi.Networks[0] = "changed"
	cloned.System.Transition.Error[0] = "changed-error"
	cloned.Profile.Startram.Info.StartramServices[0] = "~bus"
	cloned.Apps.Penpai.Info.Models[0] = "model-b"
	cloned.Logs.Containers.Wireguard.Logs[0] = "changed-wireguard-log"
	cloned.Logs.System.Logs[0] = "changed-system-log"
	delete(cloned.System.Info.Drives, "sda")
	delete(cloned.System.Info.SMART, "sda")
	delete(cloned.System.Info.Usage.Disk, "sda")
	delete(cloned.Profile.Startram.Info.Regions, "us-east")

	originalUrbit := input.Urbits["~zod"]
	if originalUrbit.Info.RemoteTlonBackups[0].Timestamp != 1 {
		t.Fatalf("remote backup slice was mutated in original state: %+v", originalUrbit.Info.RemoteTlonBackups)
	}
	if originalUrbit.Info.LocalDailyTlonBackups[0].Timestamp != 2 {
		t.Fatalf("daily backup slice was mutated in original state: %+v", originalUrbit.Info.LocalDailyTlonBackups)
	}
	if originalUrbit.Info.LocalWeeklyTlonBackups[0].Timestamp != 3 {
		t.Fatalf("weekly backup slice was mutated in original state: %+v", originalUrbit.Info.LocalWeeklyTlonBackups)
	}
	if originalUrbit.Info.LocalMonthlyTlonBackups[0].Timestamp != 4 {
		t.Fatalf("monthly backup slice was mutated in original state: %+v", originalUrbit.Info.LocalMonthlyTlonBackups)
	}
	if input.System.Info.Usage.RAM[0] != 100 {
		t.Fatalf("RAM slice was mutated in original state: %+v", input.System.Info.Usage.RAM)
	}
	if input.System.Info.Wifi.Networks[0] != "test-net" {
		t.Fatalf("wifi networks slice was mutated in original state: %+v", input.System.Info.Wifi.Networks)
	}
	if input.System.Transition.Error[0] != "first" {
		t.Fatalf("system transition errors were mutated in original state: %+v", input.System.Transition.Error)
	}
	if input.Profile.Startram.Info.StartramServices[0] != "~zod" {
		t.Fatalf("startram services were mutated in original state: %+v", input.Profile.Startram.Info.StartramServices)
	}
	if input.Apps.Penpai.Info.Models[0] != "model-a" {
		t.Fatalf("penpai models were mutated in original state: %+v", input.Apps.Penpai.Info.Models)
	}
	if input.Logs.Containers.Wireguard.Logs[0] != "wireguard-log" {
		t.Fatalf("wireguard logs were mutated in original state: %+v", input.Logs.Containers.Wireguard.Logs)
	}
	if input.Logs.System.Logs[0] != "system-log" {
		t.Fatalf("system logs were mutated in original state: %+v", input.Logs.System.Logs)
	}
	if _, exists := input.System.Info.Drives["sda"]; !exists {
		t.Fatal("drives map mutation leaked back to original state")
	}
	if _, exists := input.System.Info.SMART["sda"]; !exists {
		t.Fatal("SMART map mutation leaked back to original state")
	}
	if _, exists := input.System.Info.Usage.Disk["sda"]; !exists {
		t.Fatal("disk usage map mutation leaked back to original state")
	}
	if _, exists := input.Profile.Startram.Info.Regions["us-east"]; !exists {
		t.Fatal("regions map mutation leaked back to original state")
	}
}

func TestGetStateReturnsClone(t *testing.T) {
	original := GetState()
	t.Cleanup(func() {
		UpdateBroadcast(original)
	})

	state := structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"~zod": {},
		},
	}
	urbit := state.Urbits["~zod"]
	urbit.Info.RemoteTlonBackups = []structs.BackupObject{{Timestamp: 1}}
	state.Urbits["~zod"] = urbit
	state.System.Info.Usage.RAM = []uint64{1}
	UpdateBroadcast(state)

	copyOne := GetState()
	mutated := copyOne.Urbits["~zod"]
	mutated.Info.RemoteTlonBackups[0].Timestamp = 99
	copyOne.Urbits["~zod"] = mutated
	copyOne.System.Info.Usage.RAM[0] = 42

	copyTwo := GetState()
	gotUrbit := copyTwo.Urbits["~zod"]
	if gotUrbit.Info.RemoteTlonBackups[0].Timestamp != 1 {
		t.Fatalf("expected stored state to remain unchanged, got %+v", gotUrbit.Info.RemoteTlonBackups)
	}
	if copyTwo.System.Info.Usage.RAM[0] != 1 {
		t.Fatalf("expected stored RAM usage to remain unchanged, got %+v", copyTwo.System.Info.Usage.RAM)
	}
}

func TestGetStateJSONInjectsStructureMetadata(t *testing.T) {
	data, err := GetStateJson(structs.AuthBroadcast{})
	if err != nil {
		t.Fatalf("GetStateJson returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to decode state json: %v", err)
	}
	if decoded["type"] != "structure" {
		t.Fatalf("expected type=structure, got %v", decoded["type"])
	}
	if decoded["auth_level"] != "authorized" {
		t.Fatalf("expected auth_level=authorized, got %v", decoded["auth_level"])
	}
}

func TestGetStateJsonReturnsContextualErrorOnMarshalFailure(t *testing.T) {
	badBroadcast := structs.AuthBroadcast{}
	badUrbit := structs.Urbit{}
	badUrbit.Info.Vere = func() {}
	badBroadcast.Urbits = map[string]structs.Urbit{
		"~zod": badUrbit,
	}

	_, err := GetStateJson(badBroadcast)
	if err == nil {
		t.Fatal("expected json marshal error for invalid broadcast payload")
	}
	if !strings.Contains(err.Error(), "marshalling broadcast state payload") {
		t.Fatalf("expected contextual error message, got %v", err)
	}
	var unsupportedTypeErr *json.UnsupportedTypeError
	if !errors.As(err, &unsupportedTypeErr) {
		t.Fatalf("expected wrapped json unsupported type error, got %v", err)
	}
}

func TestSetStartramRunningUsesTransitionPath(t *testing.T) {
	previousState := GetState()
	t.Cleanup(func() {
		UpdateBroadcast(previousState)
	})

	initial := structs.AuthBroadcast{}
	initial.Profile.Startram.Info.Running = false
	initial.Profile.Startram.Transition.Restart = "running"
	UpdateBroadcast(initial)
	if err := SetStartramRunning(true); err != nil {
		t.Fatalf("SetStartramRunning returned error: %v", err)
	}
	got := GetState()
	if !got.Profile.Startram.Info.Running {
		t.Fatal("expected startram running state to be true after transition")
	}
	if got.Profile.Startram.Transition.Restart != "running" {
		t.Fatalf("expected existing startram transition state to remain, got %q", got.Profile.Startram.Transition.Restart)
	}
}
