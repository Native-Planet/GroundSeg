package restore

import (
	"fmt"
	"strings"

	"groundseg/click/internal/runtime"

	"go.uber.org/zap"
)

type RestoreRuntime interface {
	ExecuteCommandWithSuccess(string, string, string, string, string, string, func(string)) (string, error)
	RestoreAgent(string, string) error
}

type restoreRuntime struct {
	executeCommandForRestore func(string, string, string, string, string, string, func(string)) (string, error)
}

func (runtime restoreRuntime) ExecuteCommandWithSuccess(patp, file, hoon, sourcePath, successToken, operation string, clearLusCode func(string)) (string, error) {
	return runtime.executeCommandForRestore(patp, file, hoon, sourcePath, successToken, operation, clearLusCode)
}

func (runtime restoreRuntime) RestoreAgent(patp, agent string) error {
	file := fmt.Sprintf("restore-%s", agent)
	hoon := strings.Join([]string{
		"=/", "m", "(strand ,vase)",
		"^-", "form:m",
		";<", "egg=@", "bind:m", fmt.Sprintf("(scry @ /cx/base/bak/backup-%s/jam)", agent),
		";<", "our=@p", "bind:m", "get-our",
		"=/", "dk=dock", fmt.Sprintf("[our %%%s]", agent),
		"=/", "cg=cage", "[%noun !>((cue egg))]",
		"=/", "=card:agent:gall", "[%pass /pokeas %agent dk %poke %egg-any -:!>(*egg-any:gall) (cue egg)]",
		";<", "~", "bind:m", "(send-raw-card card)",
		";<", "~", "bind:m", "(take-poke-ack /pokeas)",
		"(pure:m !>('success'))",
	}, " ")
	_, err := runtime.executeCommandForRestore(patp, file, hoon, "", "success", fmt.Sprintf("Click %s", file), nil)
	if err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("Click %s restored agent %s on %s", file, agent, patp))
	return nil
}

func defaultRestoreRuntime() restoreRuntime {
	return restoreRuntime{
		executeCommandForRestore: runtime.ExecuteCommandWithSuccess,
	}
}

var runtimeRestore RestoreRuntime = defaultRestoreRuntime()

// SetRuntime replaces the internal restore runtime used by restore helpers.
func SetRuntime(handler RestoreRuntime) {
	if handler == nil {
		runtimeRestore = defaultRestoreRuntime()
		return
	}
	runtimeRestore = handler
}

func resetRestoreRuntime() {
	SetRuntime(nil)
}

func getRestoreRuntime() RestoreRuntime {
	return runtimeRestore
}

func restoreAgent(patp, agent string) error {
	return runtimeRestore.RestoreAgent(patp, agent)
}

func RestoreAgent(patp, agent string) error {
	return restoreAgent(patp, agent)
}

func RestoreTlon(patp string) error {
	var errors []string
	components := []struct {
		name string
		err  error
	}{
		{"activity", restoreAgent(patp, "activity")},
		{"channels", restoreAgent(patp, "channels")},
		{"channels-server", restoreAgent(patp, "channels-server")},
		{"groups", restoreAgent(patp, "groups")},
		{"profile", restoreAgent(patp, "profile")},
		{"chat", restoreAgent(patp, "chat")},
	}

	for _, component := range components {
		if component.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", component.name, component.err))
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("restore errors: %s", strings.Join(errors, ", "))
}
