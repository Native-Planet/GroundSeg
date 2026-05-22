package docker

import (
	"reflect"
	"testing"

	"groundseg/structs"
)

func TestBuildUrbitBootCommandUsesAssignedAmesPortInLocalMode(t *testing.T) {
	shipConf := structs.UrbitDocker{
		PierName:  "zod",
		Network:   "none",
		AmesPort:  34344,
		LoomSize:  31,
		SnapTime:  60,
		ExtraArgs: "",
	}

	command, err := BuildUrbitBootCommand(shipConf, structs.SysConfig{}, "boot")
	if err != nil {
		t.Fatalf("failed to build boot command: %v", err)
	}

	expectedArgs := []string{
		"bash",
		"/urbit/start_urbit.sh",
		"--loom=31",
		"--dirname=zod",
		"--devmode=False",
		"--http-port=80",
		"--port=34344",
		"--snap-time=60",
	}
	if !reflect.DeepEqual(command.ScriptArgs, expectedArgs) {
		t.Fatalf("unexpected script args:\n got %#v\nwant %#v", command.ScriptArgs, expectedArgs)
	}

	if command.PreviewBase != "urbit -p 34344 --http-port 80 --loom 31 --snap-time 60 zod" {
		t.Fatalf("unexpected preview: %s", command.PreviewBase)
	}
}

func TestBuildUrbitBootCommandKeepsDefaultLocalAmesPortImplicit(t *testing.T) {
	shipConf := structs.UrbitDocker{
		PierName: "zod",
		Network:  "none",
		LoomSize: 31,
		SnapTime: 60,
	}

	command, err := BuildUrbitBootCommand(shipConf, structs.SysConfig{}, "boot")
	if err != nil {
		t.Fatalf("failed to build boot command: %v", err)
	}

	expectedArgs := []string{
		"bash",
		"/urbit/start_urbit.sh",
		"--loom=31",
		"--dirname=zod",
		"--devmode=False",
		"--snap-time=60",
	}
	if !reflect.DeepEqual(command.ScriptArgs, expectedArgs) {
		t.Fatalf("unexpected script args:\n got %#v\nwant %#v", command.ScriptArgs, expectedArgs)
	}

	if command.PreviewBase != "urbit -p 34343 --http-port 80 --loom 31 --snap-time 60 zod" {
		t.Fatalf("unexpected preview: %s", command.PreviewBase)
	}
}

func TestLocalUrbitPortBindingsUseAssignedAmesPort(t *testing.T) {
	portMap, exposedPorts := localUrbitPortBindings(structs.UrbitDocker{
		HTTPPort: 8081,
		AmesPort: 34344,
	})

	amesBindings, exists := portMap["34344/udp"]
	if !exists {
		t.Fatalf("expected assigned ames container port to be bound, got %+v", portMap)
	}
	if len(amesBindings) != 1 {
		t.Fatalf("expected one ames binding, got %d", len(amesBindings))
	}
	if amesBindings[0].HostPort != "34344" {
		t.Fatalf("expected assigned host ames port 34344, got %q", amesBindings[0].HostPort)
	}
	if _, exists := portMap["34343/udp"]; exists {
		t.Fatalf("did not expect fallback ames port binding when assigned port is set")
	}
	if _, exists := exposedPorts["34344/udp"]; !exists {
		t.Fatalf("expected assigned ames port to be exposed, got %+v", exposedPorts)
	}
}
