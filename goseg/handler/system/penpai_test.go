package system

import (
	"encoding/json"
	"errors"
	"runtime"
	"testing"

	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/structs"
)

func resetPenpaiSeams() {
	confForPenpai = config.Conf
	stopContainerByNameForPenpai = orchestration.StopContainerByName
	startContainerForPenpai = orchestration.StartContainer
	updateContainerStateForPenpai = config.UpdateContainerState
	updateConfTypedForPenpai = config.UpdateConfTyped
	withPenpaiRunningForPenpai = config.WithPenpaiRunning
	withPenpaiActiveForPenpai = config.WithPenpaiActive
	withPenpaiCoresForPenpai = config.WithPenpaiCores
	deleteContainerForPenpai = orchestration.DeleteContainer
	numCPUForPenpai = runtime.NumCPU
}

func penpaiMessage(t *testing.T, action string, model string, cores int) []byte {
	t.Helper()
	msg, err := json.Marshal(structs.WsPenpaiPayload{
		Payload: structs.WsPenpaiAction{
			Action: action,
			Model:  model,
			Cores:  cores,
		},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return msg
}

func TestPenpaiHandlerValidationAndToggle(t *testing.T) {
	t.Cleanup(resetPenpaiSeams)

	if err := PenpaiHandler([]byte("{invalid")); err == nil {
		t.Fatalf("expected unmarshal error")
	}

	confForPenpai = func() structs.SysConfig { return structs.SysConfig{PenpaiRunning: false} }
	started := false
	startContainerForPenpai = func(name, typ string) (structs.ContainerState, error) {
		if name == "llama-gpt-api" && typ == "llama-api" {
			started = true
		}
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	updatedState := false
	updateContainerStateForPenpai = func(name string, _ structs.ContainerState) {
		if name == "llama-api" {
			updatedState = true
		}
	}
	confUpdates := 0
	updateConfTypedForPenpai = func(...config.ConfUpdateOption) error {
		confUpdates++
		return nil
	}
	if err := PenpaiHandler(penpaiMessage(t, "toggle", "", 0)); err != nil {
		t.Fatalf("toggle start path failed: %v", err)
	}
	if !started || !updatedState || confUpdates != 1 {
		t.Fatalf("expected start/update/config flow, got started=%v updated=%v confUpdates=%d", started, updatedState, confUpdates)
	}

	confForPenpai = func() structs.SysConfig { return structs.SysConfig{PenpaiRunning: true} }
	stopped := map[string]int{}
	stopContainerByNameForPenpai = func(name string) error {
		stopped[name]++
		return nil
	}
	if err := PenpaiHandler(penpaiMessage(t, "toggle", "", 0)); err != nil {
		t.Fatalf("toggle stop path failed: %v", err)
	}
	if stopped["llama-gpt-api"] != 1 || stopped["llama-gpt-ui"] != 1 {
		t.Fatalf("expected both containers to stop once, got %+v", stopped)
	}
}

func TestPenpaiHandlerSetModelAndSetCores(t *testing.T) {
	t.Cleanup(resetPenpaiSeams)

	confForPenpai = func() structs.SysConfig { return structs.SysConfig{PenpaiRunning: true} }
	updateConfCalls := 0
	updateConfTypedForPenpai = func(...config.ConfUpdateOption) error {
		updateConfCalls++
		return nil
	}
	deleted := 0
	deleteContainerForPenpai = func(name string) error {
		if name != "llama-gpt-api" {
			t.Fatalf("unexpected delete target: %s", name)
		}
		deleted++
		return nil
	}
	restarted := 0
	startContainerForPenpai = func(name, typ string) (structs.ContainerState, error) {
		if name == "llama-gpt-api" && typ == "llama-api" {
			restarted++
		}
		return structs.ContainerState{}, nil
	}

	if err := PenpaiHandler(penpaiMessage(t, "set-model", "phi.gguf", 0)); err != nil {
		t.Fatalf("set-model failed: %v", err)
	}
	if updateConfCalls != 1 || deleted != 1 || restarted != 1 {
		t.Fatalf("unexpected set-model flow: update=%d delete=%d restart=%d", updateConfCalls, deleted, restarted)
	}

	numCPUForPenpai = func() int { return 8 }
	if err := PenpaiHandler(penpaiMessage(t, "set-cores", "", 0)); err == nil {
		t.Fatalf("expected zero-core validation error")
	}
	if err := PenpaiHandler(penpaiMessage(t, "set-cores", "", 8)); err == nil {
		t.Fatalf("expected max-core validation error")
	}
	if err := PenpaiHandler(penpaiMessage(t, "set-cores", "", 4)); err != nil {
		t.Fatalf("set-cores valid path failed: %v", err)
	}
	if updateConfCalls != 2 || deleted != 2 || restarted != 2 {
		t.Fatalf("unexpected set-cores flow: update=%d delete=%d restart=%d", updateConfCalls, deleted, restarted)
	}
}

func TestPenpaiHandlerPropagatesOperationalErrors(t *testing.T) {
	t.Cleanup(resetPenpaiSeams)
	confForPenpai = func() structs.SysConfig { return structs.SysConfig{PenpaiRunning: true} }

	stopContainerByNameForPenpai = func(string) error { return errors.New("stop failed") }
	if err := PenpaiHandler(penpaiMessage(t, "toggle", "", 0)); err == nil {
		t.Fatalf("expected stop error propagation")
	}

	stopContainerByNameForPenpai = func(string) error { return nil }
	updateConfTypedForPenpai = func(...config.ConfUpdateOption) error { return nil }
	deleteContainerForPenpai = func(string) error { return errors.New("delete failed") }
	if err := PenpaiHandler(penpaiMessage(t, "set-model", "phi.gguf", 0)); err == nil {
		t.Fatalf("expected delete error propagation")
	}
}
