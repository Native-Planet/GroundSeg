package restore

import (
	"fmt"
	"strings"

	"groundseg/click/internal/runtime"

	"go.uber.org/zap"
)

var (
	executeClickCommandForRestore = runtime.ExecuteCommandWithSuccess
	restoreAgentFn                = restoreAgent
)

func restoreAgent(patp, agent string) error {
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
	_, err := executeClickCommandForRestore(patp, file, hoon, "", "success", fmt.Sprintf("Click %s", file), nil)
	if err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("Click %s restored agent %s on %s", file, agent, patp))
	return nil
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
		{"activity", restoreAgentFn(patp, "activity")},
		{"channels", restoreAgentFn(patp, "channels")},
		{"channels-server", restoreAgentFn(patp, "channels-server")},
		{"groups", restoreAgentFn(patp, "groups")},
		{"profile", restoreAgentFn(patp, "profile")},
		{"chat", restoreAgentFn(patp, "chat")},
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
