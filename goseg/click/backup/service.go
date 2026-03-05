package backup

import (
	"errors"
	"fmt"
	"groundseg/click/internal/runtime"
)

type BackupRuntime interface {
	ExecuteCommandWithSuccess(string, string, string, string, string, string, func(string)) (string, error)
	JoinGap([]string) string
	BackupAgent(string, string) error
}

type backupRuntime struct {
	executeClickCommandForBackup func(string, string, string, string, string, string, func(string)) (string, error)
	joinGapForBackup             func([]string) string
}

func (runtime backupRuntime) ExecuteCommandWithSuccess(patp, file, hoon, sourcePath, successToken, operation string, clearLusCode func(string)) (string, error) {
	return runtime.executeClickCommandForBackup(patp, file, hoon, sourcePath, successToken, operation, clearLusCode)
}

func (runtime backupRuntime) JoinGap(parts []string) string {
	return runtime.joinGapForBackup(parts)
}

func (runtime backupRuntime) BackupAgent(patp, agent string) error {
	file := fmt.Sprintf("backup-%s", agent)
	stateJam := "(jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))"
	scry := fmt.Sprintf("(scry egg-any:gall /gv/%s/$)", agent)
	hoon := runtime.JoinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		";<", "a=egg-any:gall", "bind:m", scry,
		";<", "~", "bind:m", fmt.Sprintf("(poke [our %%hood] %%drum-put !>([/%s/jam %s]))", file, stateJam),
		"(pure:m !>('success'))",
	})
	_, err := runtime.ExecuteCommandWithSuccess(patp, file, hoon, "", "success", fmt.Sprintf("Click %s", file), nil)
	if err != nil {
		return fmt.Errorf("click command failed for %s on %s: %w", file, patp, err)
	}
	return nil
}

func defaultBackupRuntime() backupRuntime {
	return backupRuntime{
		executeClickCommandForBackup: runtime.ExecuteCommandWithSuccess,
		joinGapForBackup:             runtime.JoinGap,
	}
}

var runtimeBackup BackupRuntime = defaultBackupRuntime()

// SetRuntime replaces the internal backup runtime used by backup helper functions.
func SetRuntime(handler BackupRuntime) {
	if handler == nil {
		runtimeBackup = defaultBackupRuntime()
		return
	}
	runtimeBackup = handler
}

func resetBackupRuntime() {
	SetRuntime(nil)
}

func getBackupRuntime() BackupRuntime {
	return runtimeBackup
}

func backupAgent(patp, agent string) error {
	return runtimeBackup.BackupAgent(patp, agent)
}

func BackupTlon(patp string) error {
	var errs []error

	components := []struct {
		name string
		err  error
	}{
		{"activity", backupAgent(patp, "activity")},
		{"channels", backupAgent(patp, "channels")},
		{"channels-server", backupAgent(patp, "channels-server")},
		{"groups", backupAgent(patp, "groups")},
		{"profile", backupAgent(patp, "profile")},
		{"chat", backupAgent(patp, "chat")},
	}

	for _, component := range components {
		if component.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", component.name, component.err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
