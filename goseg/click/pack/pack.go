package pack

import "groundseg/click/internal/runtime"

type PackRuntime interface {
	ExecuteCommand(string, string, string, string, string, string) (string, error)
}

type packRuntime struct {
	executeClickCommandForPack func(string, string, string, string, string, string) (string, error)
}

func (runtime packRuntime) ExecuteCommand(patp, file, hoon, sourcePath, successToken, operation string) (string, error) {
	return runtime.executeClickCommandForPack(patp, file, hoon, sourcePath, successToken, operation)
}

func defaultPackRuntime() packRuntime {
	return packRuntime{executeClickCommandForPack: runtime.ExecuteCommand}
}

var runtimePack PackRuntime = defaultPackRuntime()

// SetRuntime replaces the internal pack runtime used by SendPack.
func SetRuntime(handler PackRuntime) {
	if handler == nil {
		runtimePack = defaultPackRuntime()
		return
	}
	runtimePack = handler
}

func resetPackRuntime() {
	SetRuntime(nil)
}

func getPackRuntime() PackRuntime {
	return runtimePack
}

func SendPack(patp string) error {
	file := "pack"
	hoon := "=/  m  (strand ,vase)  ;<  ~  bind:m  (flog [%pack ~])  (pure:m !>('success'))"
	_, err := runtimePack.ExecuteCommand(patp, file, hoon, "", "success", "Click |pack")
	if err != nil {
		return err
	}
	return nil
}
